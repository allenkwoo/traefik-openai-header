package traefik_openai_header

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

// The RoundTripper contract lets the transport Close the request body from one
// goroutine while its write loop still Reads it, so Read and Close can reach
// free() concurrently. Both must not release the in-flight budget.
func TestBufferedBody_ConcurrentReadAndCloseReleaseBudgetOnce(t *testing.T) {
	cfg := CreateConfig()
	cfg.MaxInFlightBytes = 1 << 20
	h := newHandler(t, cfg, &nextRecorder{})
	e, ok := h.(*Handler)
	if !ok {
		t.Fatalf("New returned %T, want *Handler", h)
	}

	for i := 0; i < 500; i++ {
		data := bytes.Repeat([]byte("a"), 1024)
		held := int64(len(data))
		if !e.acquire(held) {
			t.Fatalf("iteration %d: acquire rejected within budget", i)
		}
		body := &bufferedBody{handler: e, data: data, reader: bytes.NewReader(data), held: held}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = io.Copy(io.Discard, body) }()
		go func() { defer wg.Done(); _ = body.Close() }()
		wg.Wait()

		if got := e.inFlightBytes(); got != 0 {
			t.Fatalf("iteration %d: in-flight budget = %d, want 0 (released more than once)", i, got)
		}
	}
}
