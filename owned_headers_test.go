package traefik_openai_header

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// On the requests it handles, the plugin owns the headers it derives from the
// body. A client must not be able to supply them itself, because anything
// downstream that reads them -- routing rules, rate-limit tiers, accounting --
// treats a non-empty value as plugin-derived. A body that fails to parse is
// the case that matters: the plugin sets nothing, so without clearing, the
// caller's value is what survives.

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
