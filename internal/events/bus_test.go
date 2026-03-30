package events

import (
	"sync"
	"testing"
	"time"

	"nyx/internal/domain"
)

func TestBusPublishSubscribe(t *testing.T) {
	bus := NewBus()
	ch, cancel := bus.Subscribe("flow-1")
	defer cancel()

	bus.Publish(domain.Event{FlowID: "flow-1", Type: "test", Message: "hello"})

	select {
	case evt := <-ch:
		if evt.Message != "hello" {
			t.Fatalf("expected hello, got %s", evt.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBusIsolation(t *testing.T) {
	bus := NewBus()
	ch1, cancel1 := bus.Subscribe("flow-1")
	defer cancel1()
	ch2, cancel2 := bus.Subscribe("flow-2")
	defer cancel2()

	bus.Publish(domain.Event{FlowID: "flow-1", Type: "test", Message: "for-1"})

	select {
	case <-ch2:
		t.Fatal("flow-2 subscriber should not receive flow-1 event")
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case evt := <-ch1:
		if evt.Message != "for-1" {
			t.Fatalf("expected for-1, got %s", evt.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestBusCancelUnsubscribes(t *testing.T) {
	bus := NewBus()
	ch, cancel := bus.Subscribe("flow-1")
	cancel()

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel after cancel")
	}

	// Publishing should not panic
	bus.Publish(domain.Event{FlowID: "flow-1", Type: "test", Message: "after-cancel"})
}

func TestBusConcurrentPublishCancel(t *testing.T) {
	bus := NewBus()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, cancel := bus.Subscribe("flow-race")
			defer cancel()
			// Drain a few events
			for range 3 {
				select {
				case <-ch:
				case <-time.After(10 * time.Millisecond):
				}
			}
		}()
	}

	for i := 0; i < 200; i++ {
		go func() {
			bus.Publish(domain.Event{FlowID: "flow-race", Type: "test", Message: "concurrent"})
		}()
	}

	wg.Wait()
}

func TestBusDropsWhenFull(t *testing.T) {
	bus := NewBus()
	ch, cancel := bus.Subscribe("flow-1")
	defer cancel()

	// Fill the buffer (16 cap)
	for i := 0; i < 20; i++ {
		bus.Publish(domain.Event{FlowID: "flow-1", Type: "test", Message: "fill"})
	}

	// Should have 16 buffered, 4 dropped
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 16 {
		t.Fatalf("expected 16 buffered events, got %d", count)
	}
}
