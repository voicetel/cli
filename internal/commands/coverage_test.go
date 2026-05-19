package commands

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// catchAllStub returns 200 with an empty success envelope for any request
// not explicitly stubbed. We use it to exercise every command path so
// each closure is at least entered.
func (h *testHarness) installCatchAll() {
	h.mu.routes["*"] = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"status":"success","data":{}}`)
	}
}

func (m *muxLike) catchAll() func(http.ResponseWriter, *http.Request) {
	return m.routes["*"]
}

// We extend ServeHTTP to consult the catch-all if no explicit route
// matches. (This is fine — ServeHTTP is method on muxLike defined in
// the dispatch test.)

func TestCommandSmoke(t *testing.T) {
	cases := []struct {
		method string
		path   string
		body   string
		line   string
	}{
		// account
		{"GET", "/v2.2/account/credits", `{"status":"success","data":{"credits":[]}}`, "account credits"},
		{"GET", "/v2.2/account/recurring-charges", `{"status":"success","data":{"charges":[],"total":0}}`, "account recurring-charges"},
		{"GET", "/v2.2/account/payments", `{"status":"success","data":{"payments":[]}}`, "account payments"},
		{"GET", "/v2.2/account/registration", `{"status":"success","data":{}}`, "account registration"},
		{"POST", "/v2.2/account/recovery", `{"status":"success","data":{"message":"sent"}}`, `account recover {"email":"a@b.c"}`},
		{"POST", "/v2.2/accounts", `{"status":"success","data":{"username":"u"}}`, `account signup {"name":"n","email":"a@b.c"}`},
		{"POST", "/v2.2/account", `{"status":"success","data":{"username":"u"}}`, `account add {"username":1000,"name":"n","email":"a@b.c"}`},
		{"GET", "/v2.2/account/cdr", `{"status":"success","data":{"cdr":[],"start":0,"end":0}}`, "account cdr 1700000000 1700000100"},

		// acl
		{"GET", "/v2.2/acl", `{"status":"success","data":{"acl":[]}}`, "acl list"},
		{"DELETE", "/v2.2/acl", `{"status":"success","data":{"removed":[]}}`, `acl remove {"acl":[{"cidr":"203.0.113.0/24"}]}`},

		// authentication
		{"GET", "/v2.2/auth", `{"status":"success","data":{"authType":0,"authTypeDescription":"Digest","acl":[]}}`, "authentication get"},
		{"PUT", "/v2.2/auth", `{"status":"success","data":{"updated":[]}}`, `authentication update {"authType":1}`},

		// e911
		{"GET", "/v2.2/e911", `{"status":"success","data":{"records":[]}}`, "e911 list"},
		{"POST", "/v2.2/e911", `{"status":"success","data":{"record":{}}}`, `e911 create {"dn":"2015551234","callername":"X","address1":"a","city":"c","state":"NJ","zip":"07000"}`},
		{"POST", "/v2.2/e911/validations", `{"status":"success","data":{"address":{"addressid":1}}}`, `e911 validate {"address1":"a","city":"c","state":"NJ","zip":"07000"}`},
		{"GET", "/v2.2/e911/2015551234", `{"status":"success","data":{"record":{}}}`, "e911 get 2015551234"},
		{"PUT", "/v2.2/e911/2015551234", `{"status":"success","data":{"record":{}}}`, `e911 provision 2015551234 {"callername":"X","addressid":1}`},

		// gateways
		{"GET", "/v2.2/gateways", `{"status":"success","data":{"gateways":[]}}`, "gateways list"},
		{"POST", "/v2.2/gateways", `{"status":"success","data":{"id":1}}`, `gateways add {"gateway":"203.0.113.10"}`},
		{"GET", "/v2.2/gateways/7", `{"status":"success","data":{"id":7}}`, "gateways get 7"},
		{"PUT", "/v2.2/gateways/7", `{"status":"success","data":{"id":7}}`, `gateways update 7 {"limit":50}`},
		{"DELETE", "/v2.2/gateways/7", `{"status":"success","data":null}`, "gateways remove 7"},
		{"GET", "/v2.2/gateways/7/numbers", `{"status":"success","data":{"numbers":[]}}`, "gateways numbers 7"},

		// inumbering
		{"GET", "/v2.2/inventory/coverage", `{"status":"success","data":{"coverage":[]}}`, "inumbering coverage --state=NJ"},
		{"POST", "/v2.2/orders", `{"status":"success","data":{"orderId":"o","amountCharged":0,"numbersOrdered":[]}}`, `inumbering order {"numbers":[{"Value":"2015551234"}]}`},
		{"GET", "/v2.2/ports", `{"status":"success","data":{"ports":[]}}`, "inumbering ports"},
		{"GET", "/v2.2/ports/3", `{"status":"success","data":{"port":{}}}`, "inumbering port 3"},
		{"GET", "/v2.2/ports/availability/2015551234", `{"status":"success","data":{"number":"2015551234","portable":true}}`, "inumbering port-availability 2015551234"},
		{"POST", "/v2.2/ports", `{"status":"success","data":{"pid":"X"}}`, `inumbering submit-port {"did":["2015551234"],"name":"n","nameType":"business","lcBtn":"x","lcAccountNumber":"x","streetNumber":"1","street":"Main","streetType":"ST","city":"X","state":"NJ","zip":"07000","country":"US","authPerson":"X"}`},

		// lookups
		{"GET", "/v2.2/lrn/2015551234/2125551234", `{"status":"success","data":{"ani":"a","destination":"d","lrn":{}}}`, "lookups lrn 2015551234 2125551234"},

		// messaging
		{"POST", "/v2.2/messages", `{"status":"success","data":{"id":"x","type":"sms","fromNumber":"a","toNumber":"b","parts":1}}`, `messaging send {"fromNumber":"2015551234","toNumber":"2015555678","text":"hi"}`},
		{"POST", "/v2.2/messaging/brands", `{"status":"success","data":{"result":{"statusCode":"200","status":"Success"}}}`, `messaging create-brand {"messagingBrandId":"B1","messagingBrandName":"Acme"}`},
		{"GET", "/v2.2/messaging/campaigns", `{"status":"success","data":{"campaigns":[]}}`, "messaging campaign-status"},
		{"POST", "/v2.2/messaging/campaigns", `{"status":"success","data":{"result":{"statusCode":"200","status":"Success"}}}`, `messaging create-campaign {"messagingBrandId":"B1","externalCampaignId":"E","campaignDescription":"d"}`},
		{"GET", "/v2.2/numbers/messaging", `{"status":"success","data":{"numbers":[]}}`, "messaging numbers-state --numbers=2015551234"},

		// numbers
		{"POST", "/v2.2/numbers", `{"status":"success","data":{"number":"2015551234","route":4}}`, `numbers add {"number":"2015551234"}`},
		{"GET", "/v2.2/numbers/2015551234", `{"status":"success","data":{"number":"2015551234"}}`, "numbers get 2015551234"},
		{"PATCH", "/v2.2/numbers/2015551234", `{"status":"success","data":{"number":"2015551234","accountId":1,"route":4}}`, `numbers move 2015551234 {"accountId":1,"route":4}`},
		{"POST", "/v2.2/numbers/2015551234/release", `{"status":"success","data":null}`, "numbers release 2015551234"},
		{"PUT", "/v2.2/numbers/2015551234/route", `{"status":"success","data":{"number":"2015551234","route":4}}`, `numbers set-route 2015551234 {"route":4}`},
		{"PUT", "/v2.2/numbers/2015551234/translation", `{"status":"success","data":{"number":"2015551234","translation":"1"}}`, `numbers set-translation 2015551234 {"translation":"1"}`},
		{"PUT", "/v2.2/numbers/2015551234/cnam", `{"status":"success","data":{"number":"2015551234","cnam":true}}`, `numbers set-cnam 2015551234 {"enabled":true}`},
		{"PUT", "/v2.2/numbers/2015551234/lidb", `{"status":"success","data":{"number":"2015551234","cnam":"X"}}`, `numbers set-lidb 2015551234 {"cnam":"X"}`},
		{"GET", "/v2.2/numbers/2015551234/fax", `{"status":"success","data":{"number":"2015551234","email":"a@b.c"}}`, "numbers get-fax 2015551234"},
		{"PUT", "/v2.2/numbers/2015551234/fax", `{"status":"success","data":{"number":"2015551234","email":"a@b.c"}}`, `numbers set-fax 2015551234 {"email":"a@b.c"}`},
		{"DELETE", "/v2.2/numbers/2015551234/fax", `{"status":"success","data":null}`, "numbers remove-fax 2015551234"},
		{"PUT", "/v2.2/numbers/2015551234/forward", `{"status":"success","data":{"number":"2015551234"}}`, `numbers set-forward 2015551234 {"destination":2125551234}`},
		{"DELETE", "/v2.2/numbers/2015551234/forward", `{"status":"success","data":null}`, "numbers remove-forward 2015551234"},
		{"GET", "/v2.2/numbers/2015551234/sms", `{"status":"success","data":{"number":"2015551234","type":"email","resource":"a@b.c"}}`, "numbers get-sms 2015551234"},
		{"PUT", "/v2.2/numbers/2015551234/sms", `{"status":"success","data":{"number":"2015551234","type":"email","resource":"a@b.c"}}`, `numbers set-sms 2015551234 {"type":"email","resource":"a@b.c"}`},
		{"DELETE", "/v2.2/numbers/2015551234/sms", `{"status":"success","data":null}`, "numbers remove-sms 2015551234"},
		{"GET", "/v2.2/numbers/2015551234/messaging", `{"status":"success","data":{"number":"2015551234","enabled":true,"carrier":17,"routeIn":1,"resource":"x"}}`, "numbers get-messaging 2015551234"},
		{"PATCH", "/v2.2/numbers/2015551234/messaging", `{"status":"success","data":{"number":"2015551234","updated":["routeIn"]}}`, `numbers patch-messaging 2015551234 {"routeIn":1}`},
		{"PUT", "/v2.2/numbers/2015551234/messaging-campaign", `{"status":"success","data":{"number":"2015551234","campaignId":"X","carrier":17}}`, `numbers assign-campaign 2015551234 {"campaignId":"ABCDEFG"}`},
		{"DELETE", "/v2.2/numbers/2015551234/messaging-campaign", `{"status":"success","data":{"number":"2015551234","campaignId":"X","unassigned":true}}`, "numbers unassign-campaign 2015551234"},
		{"PATCH", "/v2.2/numbers/2015551234/port-out-pin", `{"status":"success","data":{"number":"2015551234","portOutPin":"1234"}}`, `numbers set-port-out-pin 2015551234 {"pin":"1234"}`},

		// support
		{"GET", "/v2.2/support/tickets", `{"status":"success","data":{"tickets":[]}}`, "support list"},
		{"POST", "/v2.2/support/tickets", `{"status":"success","data":{"ticket":{}}}`, `support create {"subject":"x","message":"m"}`},
		{"PUT", "/v2.2/support/tickets/1", `{"status":"success","data":{"id":1,"status":"closed"}}`, `support update 1 {"status":"closed"}`},
		{"DELETE", "/v2.2/support/tickets/1", `{"status":"success","data":null}`, "support delete 1"},
		{"GET", "/v2.2/support/tickets/1/messages", `{"status":"success","data":{"messages":[]}}`, "support messages 1"},
		{"POST", "/v2.2/support/tickets/1/replies", `{"status":"success","data":{"message":"Reply added"}}`, `support reply 1 {"message":"hi"}`},
	}

	for _, tc := range cases {
		name := strings.ReplaceAll(strings.SplitN(tc.line, " ", 3)[0], "/", "_")
		if i := strings.IndexByte(tc.line, ' '); i >= 0 {
			name = strings.ReplaceAll(tc.line[:i+1], " ", "_") + strings.SplitN(tc.line[i+1:], " ", 2)[0]
		}
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.stub(tc.method, tc.path, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, tc.body)
			})
			r := BuildRegistry()
			if err := h.dispatch(tc.line, r); err != nil {
				t.Fatalf("%q: %v\nstderr: %s", tc.line, err, h.err.String())
			}
		})
	}
	// Reference installCatchAll to keep go vet happy if test list shrinks.
	_ = (*testHarness).installCatchAll
	_ = (*muxLike).catchAll
}

// TestErrorBranches exercises every "missing arg / bad arg / bad json"
// branch in the resource closures so error paths show up in coverage.
func TestErrorBranches(t *testing.T) {
	bad := []string{
		// Numbers: each handler that requires an arg
		"numbers get",
		"numbers remove",
		"numbers move",
		"numbers move 2015551234 not-json",
		"numbers release",
		"numbers set-route",
		"numbers set-route 2015551234 not-json",
		"numbers set-translation",
		"numbers set-translation 2015551234 not-json",
		"numbers set-cnam",
		"numbers set-cnam 2015551234 not-json",
		"numbers set-lidb",
		"numbers set-lidb 2015551234 not-json",
		"numbers get-fax",
		"numbers set-fax",
		"numbers set-fax 2015551234 not-json",
		"numbers remove-fax",
		"numbers set-forward",
		"numbers set-forward 2015551234 not-json",
		"numbers remove-forward",
		"numbers get-sms",
		"numbers set-sms",
		"numbers set-sms 2015551234 not-json",
		"numbers remove-sms",
		"numbers get-messaging",
		"numbers patch-messaging",
		"numbers patch-messaging 2015551234 not-json",
		"numbers assign-campaign",
		"numbers assign-campaign 2015551234 not-json",
		"numbers unassign-campaign",
		"numbers bulk-unassign-campaign",
		"numbers set-port-out-pin",
		"numbers set-port-out-pin 2015551234 not-json",
		"numbers add not-json",

		// Gateways
		"gateways get",
		"gateways add not-json",
		"gateways update",
		"gateways update abc {}",
		"gateways update 1 not-json",
		"gateways remove",
		"gateways remove abc",
		"gateways numbers",
		"gateways numbers abc",

		// E911
		"e911 get",
		"e911 provision",
		"e911 provision 2015551234 not-json",
		"e911 remove",
		"e911 create not-json",
		"e911 validate not-json",

		// Support
		"support get",
		"support get abc",
		"support update",
		"support update abc {}",
		"support update 1 not-json",
		"support delete",
		"support delete abc",
		"support messages",
		"support messages abc",
		"support reply",
		"support reply abc {}",
		"support reply 1 not-json",
		"support create not-json",

		// Lookups
		"lookups cnam",
		"lookups lrn",
		"lookups lrn 2015551234",

		// INumbering
		"inumbering port",
		"inumbering port abc",
		"inumbering port-availability",
		"inumbering order not-json",
		"inumbering submit-port not-json",
		"inumbering search-inventory --npa=bad",
		"inumbering search-inventory --nxx=bad",
		"inumbering search-inventory --limit=bad",
		"inumbering search-inventory --unknown=x",

		// Messaging
		"messaging send not-json",
		"messaging create-brand not-json",
		"messaging create-campaign not-json",
		"messaging history --start=bad",
		"messaging history --end=bad",

		// ACL
		"acl add not-json",
		"acl remove not-json",

		// Authentication
		"authentication update not-json",

		// Account
		"account update not-json",
		"account add not-json",
		"account signup not-json",
		"account recover not-json",
		"account cdr bad",
		"account cdr 1700000000 bad",
	}
	for _, line := range bad {
		h := newHarness(t)
		r := BuildRegistry()
		if err := h.dispatch(line, r); err == nil {
			t.Errorf("%q: expected error, got success (stdout: %s)", line, h.out.String())
		}
	}
}

// Verify "remove-sms" works (DELETE returns 204).
func TestNumbersRemoveSMS204(t *testing.T) {
	h := newHarness(t)
	h.stub("DELETE", "/v2.2/numbers/2015551234/sms", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	})
	r := BuildRegistry()
	if err := h.dispatch("numbers remove-sms 2015551234", r); err != nil {
		t.Fatal(err)
	}
}
