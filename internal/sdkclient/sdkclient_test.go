package sdkclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testUA      = "voicetel-cli-test/0.0.0"
	testKey     = "abcdef0123456789abcdef0123456789"
	testNewKey  = "1234567890abcdef1234567890abcdef"
	testBaseURL = "https://example.invalid"
)

// New + accessors: construct with each combination of inputs and verify the
// reported state matches.
func TestNewConstructsClient(t *testing.T) {
	c := New(testBaseURL, testKey, testUA)
	if got := c.BaseURL(); got != testBaseURL {
		t.Errorf("BaseURL: want %q, got %q", testBaseURL, got)
	}
	if got := c.APIKey(); got != testKey {
		t.Errorf("APIKey: want %q, got %q", testKey, got)
	}
}

func TestNewWithEmptyBaseURL(t *testing.T) {
	// Empty baseURL → SDK falls back to its compiled-in production endpoint.
	c := New("", testKey, testUA)
	if got := c.BaseURL(); got == "" {
		t.Error("BaseURL: expected SDK default, got empty")
	}
}

func TestNewWithEmptyAPIKey(t *testing.T) {
	c := New(testBaseURL, "", testUA)
	if got := c.APIKey(); got != "" {
		t.Errorf("APIKey: want empty, got %q", got)
	}
}

// SetAPIKey rebuilds the inner client; the new key is reported but the base URL
// and user agent must survive.
func TestSetAPIKeyRebuilds(t *testing.T) {
	c := New(testBaseURL, testKey, testUA)
	c.SetAPIKey(testNewKey)
	if got := c.APIKey(); got != testNewKey {
		t.Errorf("APIKey after SetAPIKey: want %q, got %q", testNewKey, got)
	}
	if got := c.BaseURL(); got != testBaseURL {
		t.Errorf("BaseURL after SetAPIKey: want %q (preserved), got %q", testBaseURL, got)
	}
}

// All 10 service accessors should return non-nil pointers wired through to the
// inner SDK.
func TestServiceAccessorsAllReturnNonNil(t *testing.T) {
	c := New(testBaseURL, testKey, testUA)
	cases := map[string]any{
		"Account":        c.Account(),
		"ACL":            c.ACL(),
		"Authentication": c.Authentication(),
		"E911":           c.E911(),
		"Gateways":       c.Gateways(),
		"INumbering":     c.INumbering(),
		"Lookups":        c.Lookups(),
		"Messaging":      c.Messaging(),
		"Numbers":        c.Numbers(),
		"Support":        c.Support(),
	}
	for name, svc := range cases {
		if svc == nil {
			t.Errorf("%s(): returned nil", name)
		}
	}
}

// Login round-trips through the SDK's Login() into the real transport. We stand
// up an httptest server that pretends to be the /account/api-key endpoint and
// returns a fresh key. The test confirms we surfaced the key to the caller.
func TestLoginExchangesForAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/account/api-key") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// Mirrors the VoiceTel API response: a JSON object with an "apikey" field.
		_ = json.NewEncoder(w).Encode(map[string]string{"apikey": testNewKey})
	}))
	defer srv.Close()

	c := New(srv.URL, "", testUA)
	key, err := c.Login(context.Background(), 1000000001, "hunter2")
	if err != nil {
		t.Fatalf("Login: unexpected error: %v", err)
	}
	if key != testNewKey {
		t.Errorf("Login: want key %q, got %q", testNewKey, key)
	}
}

// Login error: server returns 401 → SDK reports an error → wrapper surfaces it.
func TestLoginPropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "", testUA)
	if _, err := c.Login(context.Background(), 1000000001, "wrong"); err == nil {
		t.Fatal("Login: expected error from 401, got nil")
	}
}
