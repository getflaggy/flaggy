package sse

import (
	"runtime"
	"testing"
	"time"
)

func TestUnsubscribeReturnsAndReleasesClient(t *testing.T) {
	b := NewBroadcaster()
	ch, unsub := b.Subscribe()

	done := make(chan struct{})
	go func() {
		unsub()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("unsubscribe blocked")
	}

	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed after unsubscribe")
	}
	if b.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", b.ClientCount())
	}
}

func TestUnsubscribeDoesNotLeakGoroutines(t *testing.T) {
	b := NewBroadcaster()
	before := runtime.NumGoroutine()
	for i := 0; i < 200; i++ {
		_, unsub := b.Subscribe()
		unsub()
	}
	time.Sleep(50 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutines leaked: before=%d after=%d", before, after)
	}
}

func TestUnsubscribeIsIdempotentAndSafeAfterClose(t *testing.T) {
	b := NewBroadcaster()
	_, unsub := b.Subscribe()
	unsub()
	unsub()

	_, unsub2 := b.Subscribe()
	b.Close()
	unsub2()
}

func TestPublishAfterUnsubscribeDoesNotPanic(t *testing.T) {
	b := NewBroadcaster()
	_, unsub := b.Subscribe()
	kept, unsubKept := b.Subscribe()
	defer unsubKept()

	unsub()
	b.Publish(Event{Type: "flag_updated"})

	select {
	case ev := <-kept:
		if ev.Type != "flag_updated" {
			t.Fatalf("unexpected event %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("remaining subscriber did not receive event")
	}
}
