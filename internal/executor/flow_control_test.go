package executor

import (
	"context"
	"testing"
	"time"
)

func TestFlowControllerEnforcesMinIntervalPerFlow(t *testing.T) {
	controller := NewFlowController(1, 40*time.Millisecond)

	release, err := controller.Acquire(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	release()

	started := time.Now()
	release, err = controller.Acquire(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	release()

	if elapsed := time.Since(started); elapsed < 35*time.Millisecond {
		t.Fatalf("expected min-interval wait, got %s", elapsed)
	}
}

func TestFlowControllerIsolationAcrossFlows(t *testing.T) {
	controller := NewFlowController(1, 50*time.Millisecond)

	release, err := controller.Acquire(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("acquire flow-1: %v", err)
	}
	defer release()

	started := time.Now()
	releaseOther, err := controller.Acquire(context.Background(), "flow-2")
	if err != nil {
		t.Fatalf("acquire flow-2: %v", err)
	}
	defer releaseOther()

	if elapsed := time.Since(started); elapsed > 20*time.Millisecond {
		t.Fatalf("expected different flow to bypass rate wait, got %s", elapsed)
	}
}
