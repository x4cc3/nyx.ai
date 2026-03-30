package executor

import (
	"context"
	"sync"
	"time"
)

type FlowController struct {
	maxConcurrent int
	minInterval   time.Duration
	retryBackoff  time.Duration

	mu     sync.Mutex
	states map[string]*flowControlState
}

type flowControlState struct {
	active      int
	lastStarted time.Time
}

func NewFlowController(maxConcurrent int, minInterval time.Duration) *FlowController {
	return NewFlowControllerWithBackoff(maxConcurrent, minInterval, 25*time.Millisecond)
}

func NewFlowControllerWithBackoff(maxConcurrent int, minInterval, retryBackoff time.Duration) *FlowController {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if minInterval < 0 {
		minInterval = 0
	}
	if retryBackoff <= 0 {
		retryBackoff = 25 * time.Millisecond
	}
	return &FlowController{
		maxConcurrent: maxConcurrent,
		minInterval:   minInterval,
		retryBackoff:  retryBackoff,
		states:        make(map[string]*flowControlState),
	}
}

func (c *FlowController) Acquire(ctx context.Context, flowID string) (func(), error) {
	flowID = normalizeFlowID(flowID)
	for {
		wait := c.tryAcquire(flowID)
		if wait <= 0 {
			return func() { c.release(flowID) }, nil
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *FlowController) tryAcquire(flowID string) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.state(flowID)
	now := time.Now()
	if state.active >= c.maxConcurrent {
		return c.retryBackoff
	}
	if c.minInterval > 0 {
		nextAllowed := state.lastStarted.Add(c.minInterval)
		if now.Before(nextAllowed) {
			return nextAllowed.Sub(now)
		}
	}
	state.active++
	state.lastStarted = now
	return 0
}

func (c *FlowController) release(flowID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state, ok := c.states[flowID]
	if !ok {
		return
	}
	if state.active > 0 {
		state.active--
	}
	if state.active == 0 && c.minInterval > 0 && time.Since(state.lastStarted) > c.minInterval {
		delete(c.states, flowID)
	}
}

func (c *FlowController) state(flowID string) *flowControlState {
	state, ok := c.states[flowID]
	if !ok {
		state = &flowControlState{}
		c.states[flowID] = state
	}
	return state
}

func normalizeFlowID(flowID string) string {
	if flowID == "" {
		return "default"
	}
	return flowID
}
