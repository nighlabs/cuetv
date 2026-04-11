package broker

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// SSEBroker manages session-scoped Server-Sent Events fan-out. Each session
// maintains an independent set of subscriber channels so events for one
// session are never delivered to viewers of another. The broker is safe for
// concurrent use.
type SSEBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan string]struct{} // sessionID -> set of channels
}

// NewSSEBroker creates an SSEBroker with an empty subscriber map, ready to
// accept subscriptions.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		subscribers: make(map[string]map[chan string]struct{}),
	}
}

// Subscribe registers a new viewer for the given session and returns a
// buffered channel that will receive broadcast events. The channel is
// buffered to 16 entries so that a brief consumer stall does not block the
// broadcaster; events exceeding the buffer are dropped (see Broadcast).
func (b *SSEBroker) Subscribe(sessionID string) chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan string, 16)
	if b.subscribers[sessionID] == nil {
		b.subscribers[sessionID] = make(map[chan string]struct{})
	}
	b.subscribers[sessionID][ch] = struct{}{}

	count := len(b.subscribers[sessionID])
	slog.Info("SSE subscriber connected", "sessionId", sessionID, "viewerCount", count)

	return ch
}

// Unsubscribe removes the given channel from the session's subscriber set and
// closes it. If the session has no remaining subscribers, its entry is cleaned
// up entirely.
func (b *SSEBroker) Unsubscribe(sessionID string, ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[sessionID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(b.subscribers, sessionID)
		}
	}

	count := len(b.subscribers[sessionID])
	slog.Info("SSE subscriber disconnected", "sessionId", sessionID, "viewerCount", count)
}

// Broadcast sends an event to every subscriber of the given session. The
// event data is sanitized before delivery to prevent SSE injection. Slow
// consumers whose channel buffers are full will have the event dropped rather
// than blocking the broadcaster; dropped events are logged at Warn level.
func (b *SSEBroker) Broadcast(sessionID, event string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.subscribers[sessionID]
	if !ok {
		return
	}

	sanitized := SanitizeSSEData(event)
	dropped := 0

	for ch := range subs {
		select {
		case ch <- sanitized:
		default:
			// Drop event for slow consumer rather than blocking the broadcast loop.
			dropped++
		}
	}

	if dropped > 0 {
		slog.Warn("dropped SSE events for slow consumers",
			"sessionId", sessionID,
			"dropped", dropped,
		)
	}

	slog.Debug("SSE event broadcast",
		"sessionId", sessionID,
		"event", event,
		"recipientCount", len(subs)-dropped,
	)
}

// SubscriberCount returns the number of active SSE subscribers for a session.
func (b *SSEBroker) SubscriberCount(sessionID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[sessionID])
}

// SanitizeSSEData escapes newlines to prevent SSE injection. SSE data fields
// are line-delimited, so embedded newlines would inject additional fields or
// terminate the event prematurely.
func SanitizeSSEData(data string) string {
	data = strings.ReplaceAll(data, "\n", " ")
	data = strings.ReplaceAll(data, "\r", " ")
	return data
}

// FormatSSEEvent wraps the given event string in the SSE wire format
// ("data: ...\n\n") ready to be written to an HTTP response stream.
func FormatSSEEvent(event string) string {
	return fmt.Sprintf("data: %s\n\n", event)
}
