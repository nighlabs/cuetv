package broker

import (
	"testing"
	"time"
)

func TestSSEBroker_SubscribeAndBroadcast(t *testing.T) {
	b := NewSSEBroker()

	ch := b.Subscribe("session-1")
	defer b.Unsubscribe("session-1", ch)

	b.Broadcast("session-1", "play")

	select {
	case msg := <-ch:
		if msg != "play" {
			t.Errorf("expected 'play', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSSEBroker_SessionIsolation(t *testing.T) {
	b := NewSSEBroker()

	ch1 := b.Subscribe("session-1")
	defer b.Unsubscribe("session-1", ch1)

	ch2 := b.Subscribe("session-2")
	defer b.Unsubscribe("session-2", ch2)

	b.Broadcast("session-1", "play")

	select {
	case msg := <-ch1:
		if msg != "play" {
			t.Errorf("session-1 expected 'play', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for session-1 event")
	}

	select {
	case msg := <-ch2:
		t.Errorf("session-2 should not receive event, got %q", msg)
	case <-time.After(100 * time.Millisecond):
		// Expected: no event for session-2
	}
}

func TestSSEBroker_SlowConsumerDropped(t *testing.T) {
	b := NewSSEBroker()

	ch := b.Subscribe("session-1")
	defer b.Unsubscribe("session-1", ch)

	// Fill the buffer (capacity 16)
	for i := 0; i < 20; i++ {
		b.Broadcast("session-1", "event")
	}

	// Should have received 16 events, 4 dropped
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
		t.Errorf("expected 16 buffered events, got %d", count)
	}
}

func TestSSEBroker_Unsubscribe(t *testing.T) {
	b := NewSSEBroker()

	ch := b.Subscribe("session-1")
	if b.SubscriberCount("session-1") != 1 {
		t.Errorf("expected 1 subscriber, got %d", b.SubscriberCount("session-1"))
	}

	b.Unsubscribe("session-1", ch)
	if b.SubscriberCount("session-1") != 0 {
		t.Errorf("expected 0 subscribers, got %d", b.SubscriberCount("session-1"))
	}
}

func TestSanitizeSSEData(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"play", "play"},
		{"load:abc123", "load:abc123"},
		{"evil\ndata: injected", "evil data: injected"},
		{"evil\r\ndata: injected", "evil  data: injected"},
	}

	for _, tt := range tests {
		got := SanitizeSSEData(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeSSEData(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
