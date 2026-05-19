package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/voicetel/cli/internal/output"
	"github.com/voicetel/cli/internal/sdkclient"
)

// testHarness builds a Context bound to an httptest-backed SDK client,
// captures stdout/stderr, and exposes the route map for stubbing.
type testHarness struct {
	srv     *httptest.Server
	client  sdkclient.Client
	out     *bytes.Buffer
	err     *bytes.Buffer
	context *Context

	mu     *muxLike
	cfgSet *bool // OnConfigChanged was invoked
}

type muxLike struct {
	routes map[string]func(http.ResponseWriter, *http.Request)
}

func (m *muxLike) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	if h, ok := m.routes[key]; ok {
		h(w, r)
		return
	}
	// Fallback: prefix match for paths with variable trailing segments.
	for k, h := range m.routes {
		parts := strings.SplitN(k, " ", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] != r.Method {
			continue
		}
		if strings.HasSuffix(parts[1], "*") {
			if strings.HasPrefix(r.URL.Path, strings.TrimSuffix(parts[1], "*")) {
				h(w, r)
				return
			}
		}
	}
	http.Error(w, "no stub for "+key, http.StatusNotFound)
}

func newHarness(t *testing.T) *testHarness {
	t.Helper()
	h := &testHarness{
		mu:     &muxLike{routes: map[string]func(http.ResponseWriter, *http.Request){}},
		out:    &bytes.Buffer{},
		err:    &bytes.Buffer{},
		cfgSet: new(bool),
	}
	h.srv = httptest.NewServer(h.mu)
	t.Cleanup(h.srv.Close)
	h.client = sdkclient.New(h.srv.URL, "deadbeef", "voicetel-cli-test/0.0.0")
	h.context = &Context{
		Ctx:     context.Background(),
		Client:  h.client,
		Printer: output.NewWithColor(h.out, h.err, false),
		OnConfigChanged: func() {
			*h.cfgSet = true
		},
	}
	return h
}

// stub registers an HTTP handler for METHOD PATH (path may end with "*" for prefix match).
func (h *testHarness) stub(method, path string, fn func(http.ResponseWriter, *http.Request)) {
	h.mu.routes[method+" "+path] = fn
}

// dispatch parses and runs a single command line, returning any error.
func (h *testHarness) dispatch(line string, registry *Registry) error {
	return runLine(h.context, registry, line)
}

// runLine is the dispatch helper analogous to the main package's
// dispatchLine, but living inside this package for tests.
func runLine(c *Context, r *Registry, line string) error {
	// Tokenize. Walk through whitespace, honour quotes.
	tokens, raw, err := simpleTokenize(line)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}
	return r.Dispatch(c, tokens, raw)
}

func simpleTokenize(line string) ([]string, string, error) {
	s := strings.TrimSpace(line)
	if s == "" {
		return nil, "", nil
	}
	var out []string
	cur := strings.Builder{}
	inQ := byte(0)
	piece := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case inQ != 0:
			if ch == inQ {
				inQ = 0
				continue
			}
			cur.WriteByte(ch)
			piece = true
		case ch == '"' || ch == '\'':
			inQ = ch
			piece = true
		case ch == ' ' || ch == '\t':
			if piece {
				out = append(out, cur.String())
				cur.Reset()
				piece = false
			}
		default:
			cur.WriteByte(ch)
			piece = true
		}
	}
	if inQ != 0 {
		return nil, "", errors.New("unterminated quote")
	}
	if piece {
		out = append(out, cur.String())
	}
	return out, s, nil
}

// --- tests -----------------------------------------------------------------

func TestDispatchUnknownCommand(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	err := h.dispatch("nope", r)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestDispatchHelp(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("help", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "Top-level commands") {
		t.Errorf("help didn't render registry: %s", h.out.String())
	}
}

func TestDispatchHelpGroup(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("help numbers", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "numbers list") {
		t.Errorf("help numbers didn't list: %s", h.out.String())
	}
}

func TestDispatchHelpCommand(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("help numbers list", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "Every TN on the account") {
		t.Errorf("help numbers list missing synopsis: %s", h.out.String())
	}
}

func TestDispatchHelpUnknown(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("help nope", r); err == nil {
		t.Fatal("expected error for unknown help topic")
	}
}

func TestDispatchExit(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	err := h.dispatch("exit", r)
	if !errors.Is(err, ErrExit) {
		t.Fatalf("expected ErrExit, got %v", err)
	}
}

func TestDispatchClear(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("clear", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "\x1b[2J") {
		t.Errorf("clear didn't emit ANSI clear: %q", h.out.String())
	}
}

func TestNumbersList(t *testing.T) {
	h := newHarness(t)
	h.stub("GET", "/v2.2/numbers", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"numbers":[{"number":"2015551234","route":4}]}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("numbers list", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "2015551234") {
		t.Errorf("expected number in output, got %s", h.out.String())
	}
}

func TestNumbersGetMissingArg(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("numbers get", r); err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestNumbersAddInvalidJSON(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	err := h.dispatch(`numbers add not-json`, r)
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestAccountGet(t *testing.T) {
	h := newHarness(t)
	h.stub("GET", "/v2.2/account", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"username":"1000000001","cash":12.34,"callerId":"2015551234"}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("account get", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "callerId") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestAccountUpdate(t *testing.T) {
	h := newHarness(t)
	h.stub("PUT", "/v2.2/account", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"updated":["timezone"]}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch(`account update {"timezone":"America/Chicago"}`, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "updated") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestNumbersRemove(t *testing.T) {
	h := newHarness(t)
	h.stub("DELETE", "/v2.2/numbers/2015551234", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	r := BuildRegistry()
	if err := h.dispatch("numbers remove 2015551234", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "Removed") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestGatewaysGetBadID(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("gateways get abc", r); err == nil {
		t.Fatal("expected error for non-integer id")
	}
}

func TestLookupsCNAM(t *testing.T) {
	h := newHarness(t)
	h.stub("GET", "/v2.2/cnam/2015551234", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"cnam":"ALICE COHEN","number":"2015551234"}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("lookups cnam 2015551234", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "ALICE COHEN") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestMessagingHistoryFlags(t *testing.T) {
	h := newHarness(t)
	h.stub("GET", "/v2.2/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("number") != "2015551234" {
			t.Errorf("missing number query: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("type") != "sms" {
			t.Errorf("missing type query: %s", r.URL.RawQuery)
		}
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"number":"2015551234","type":"sms","fromTs":0,"toTs":0,"messages":[]}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("messaging history --number=2015551234 --type=sms", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), `"type": "sms"`) {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestINumberingSearchInventoryFlags(t *testing.T) {
	h := newHarness(t)
	h.stub("GET", "/v2.2/inventory", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("npa") != "201" {
			t.Errorf("missing npa: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("state") != "NJ" {
			t.Errorf("missing state: %s", r.URL.RawQuery)
		}
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"numbers":[{"number":"2015551234","rateCenter":"X","city":"Y","province":"NJ","lata":"222"}]}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("inumbering search-inventory --npa=201 --state=NJ", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "2015551234") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestNumbersBulkUnassignCampaign(t *testing.T) {
	h := newHarness(t)
	h.stub("DELETE", "/v2.2/numbers/messaging-campaign", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"campaignId":"ABCDEFG","unassignedNumbers":["2015551234","2015551235"]}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("numbers bulk-unassign-campaign 2015551234,2015551235", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "ABCDEFG") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestACLAdd(t *testing.T) {
	h := newHarness(t)
	h.stub("POST", "/v2.2/acl", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"added":[{"cidr":"203.0.113.0/24"}]}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch(`acl add {"acl":[{"cidr":"203.0.113.0/24"}]}`, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "203.0.113.0/24") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestSupportGet(t *testing.T) {
	h := newHarness(t)
	h.stub("GET", "/v2.2/support/tickets/42", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"ticket":{"id":42,"status":"active","subject":"hi"}}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("support get 42", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), `"id": 42`) {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestE911Remove(t *testing.T) {
	h := newHarness(t)
	h.stub("DELETE", "/v2.2/e911/2015551234", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	r := BuildRegistry()
	if err := h.dispatch("e911 remove 2015551234", r); err != nil {
		t.Fatal(err)
	}
}

func TestWhoami(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("whoami", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "apiKey") {
		t.Errorf("got: %s", h.out.String())
	}
}

func TestLogin(t *testing.T) {
	h := newHarness(t)
	h.stub("POST", "/v2.2/account/api-key", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"apikey":"abcdef0123456789abcdef0123456789"}}`)
	})
	r := BuildRegistry()
	if err := h.dispatch("login 1000000001 hunter2", r); err != nil {
		t.Fatal(err)
	}
	if !*h.cfgSet {
		t.Error("OnConfigChanged not invoked")
	}
}

func TestLoginBadArgs(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("login alice secret", r); err == nil {
		t.Fatal("expected error: username must be int")
	}
	if err := h.dispatch("login 1000", r); err == nil {
		t.Fatal("expected error: missing password")
	}
}

func TestSetAPIKey(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("set api-key newkey", r); err != nil {
		t.Fatal(err)
	}
	if !*h.cfgSet {
		t.Error("OnConfigChanged not invoked")
	}
}

func TestSetUnknownKey(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("set unknown x", r); err == nil {
		t.Fatal("expected error")
	}
}

func TestCompletionData(t *testing.T) {
	r := BuildRegistry()
	builtins, topics, groups := r.CompletionData()
	if len(builtins) == 0 {
		t.Error("no builtins")
	}
	if len(topics) == 0 {
		t.Error("no topics")
	}
	if _, ok := groups["numbers"]; !ok {
		t.Error("numbers group missing")
	}
	var foundList bool
	for _, s := range groups["numbers"] {
		if s == "list" {
			foundList = true
		}
	}
	if !foundList {
		t.Error("numbers list missing from completions")
	}
}

func TestFindGroupAndBuiltin(t *testing.T) {
	r := BuildRegistry()
	if r.FindGroup("nope") != nil {
		t.Error("found phantom group")
	}
	if r.FindGroup("numbers") == nil {
		t.Error("missing numbers group")
	}
	if r.FindBuiltin("nope") != nil {
		t.Error("found phantom builtin")
	}
	if r.FindBuiltin("help") == nil {
		t.Error("missing help builtin")
	}
}

func TestDispatchGroupNoSub(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("numbers", r); err == nil {
		t.Fatal("expected error for missing sub-command")
	}
}

func TestDispatchUnknownSub(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("numbers nope", r); err == nil {
		t.Fatal("expected error for unknown sub")
	}
}

func TestSkipArgs(t *testing.T) {
	got := skipArgs(`2015551234 {"route":4}`, 1)
	if got != `{"route":4}` {
		t.Errorf("skipArgs = %q", got)
	}
	if skipArgs(``, 1) != "" {
		t.Error("skipArgs on empty should be empty")
	}
	if skipArgs("just-one-token", 1) != "" {
		t.Error("skipArgs consuming the whole input")
	}
	// Quote-aware
	if g := skipArgs(`"hello world" rest`, 1); g != "rest" {
		t.Errorf("quote-aware skipArgs = %q", g)
	}
}

func TestPrintResultGNilSkip(t *testing.T) {
	h := newHarness(t)
	type T struct{ Name string }
	var v *T
	if err := printResultG(h.context, v, nil); err != nil {
		t.Fatal(err)
	}
	if h.out.Len() != 0 {
		t.Errorf("expected no output for nil result, got %q", h.out.String())
	}
}

func TestPrintResultGErrorPasses(t *testing.T) {
	h := newHarness(t)
	type T struct{}
	if err := printResultG(h.context, (*T)(nil), errSentinel); err != errSentinel {
		t.Fatalf("expected sentinel, got %v", err)
	}
}

var errSentinel = stringError("sentinel")

type stringError string

func (s stringError) Error() string { return string(s) }

// Ensure the SDK adapter passes through correctly.
func TestSDKClientAdapter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"username":"u"}}`)
	}))
	defer srv.Close()
	c := sdkclient.New(srv.URL, "key", "ua")
	if c.BaseURL() != srv.URL {
		t.Errorf("base url = %q", c.BaseURL())
	}
	if c.APIKey() != "key" {
		t.Errorf("api key = %q", c.APIKey())
	}
	c.SetAPIKey("newkey")
	if c.APIKey() != "newkey" {
		t.Errorf("after set, api key = %q", c.APIKey())
	}
	// Sanity-check service accessor.
	if _, err := c.Account().Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Force a *voicetel.AccountService — just compile-time sanity.
	_ = c.Account() // compile-time sanity
}
