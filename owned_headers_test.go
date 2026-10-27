package traefik_openai_header

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The plugin owns the headers it derives from the body. A client must not be
// able to supply them itself, because anything downstream that reads them --
// routing rules, rate-limit tiers, accounting -- treats a non-empty value as
// plugin-derived.

func TestOwnedHeaders_ClientSuppliedModelClearedOnNonMatchingRoute(t *testing.T) {
	next := &nextRecorder{}
	h := newHandler(t, CreateConfig(), next)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("X-OpenAI-Model", "spoofed")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got := next.header.Get("X-OpenAI-Model"); got != "" {
		t.Errorf("X-OpenAI-Model = %q, want empty: a client-supplied value reached downstream", got)
	}
}

func TestOwnedHeaders_ClientSuppliedModelClearedOnUnparseableBody(t *testing.T) {
	next := &nextRecorder{}
	h := newHandler(t, CreateConfig(), next)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString("not json"))
	req.Header.Set("X-OpenAI-Model", "spoofed")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got := next.header.Get("X-OpenAI-Model"); got != "" {
		t.Errorf("X-OpenAI-Model = %q, want empty: a client-supplied value survived an unparseable body", got)
	}
	if got := next.header.Get(ParseFailureHeader); got == "" {
		t.Errorf("%s = empty, want a parse failure recorded", ParseFailureHeader)
	}
}

func TestOwnedHeaders_ClientSuppliedParseFailureCleared(t *testing.T) {
	next := &nextRecorder{}
	h := newHandler(t, CreateConfig(), next)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set(ParseFailureHeader, "pretend-it-parsed")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got := next.header.Get(ParseFailureHeader); got != "" {
		t.Errorf("%s = %q, want empty: a client-supplied value reached downstream", ParseFailureHeader, got)
	}
}

// Clearing must not disturb the headers the plugin legitimately derives.
func TestOwnedHeaders_DerivedModelStillReachesDownstream(t *testing.T) {
	next := &nextRecorder{}
	h := newHandler(t, CreateConfig(), next)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		bytes.NewBufferString(`{"model":"gpt-4"}`))
	req.Header.Set("X-OpenAI-Model", "spoofed")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got := next.header.Get("X-OpenAI-Model"); got != "gpt-4" {
		t.Errorf("X-OpenAI-Model = %q, want \"gpt-4\"", got)
	}
}
