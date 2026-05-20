package commands

import (
	"net/http"
	"strings"
	"testing"
)

// whoami when no API key is installed surfaces an empty / "(none)" string
// for the apiKey field — exercises the no-key branch.
func TestWhoamiNoAPIKey(t *testing.T) {
	h := newHarness(t)
	h.client.SetAPIKey("")
	r := BuildRegistry()
	if err := h.dispatch("whoami", r); err != nil {
		t.Fatal(err)
	}
	got := h.out.String()
	if !strings.Contains(got, "apiKey") {
		t.Errorf("whoami output missing apiKey label: %s", got)
	}
}

// loginCommand with a Login that errors at the SDK layer (server returns
// 401) propagates the error.
func TestLoginCommandSDKError(t *testing.T) {
	h := newHarness(t)
	h.stub("POST", "/v2.2/account/api-key", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	r := BuildRegistry()
	if err := h.dispatch("login 1000000001 hunter2", r); err == nil {
		t.Fatal("expected 401 to propagate, got nil")
	}
}

// renderCommand path for a leaf command: exercise via `help numbers list`
// and verify the output includes the command name.
func TestRenderCommandListsSubs(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("help numbers list", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "list") {
		t.Errorf("expected 'list' in help text: %s", h.out.String())
	}
}
