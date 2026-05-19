// Package sdkclient is a thin abstraction over *voicetel.Client so that
// command dispatch can be tested without making real HTTP calls.
//
// The interface intentionally mirrors only the surface area the CLI uses;
// new commands extend the interface (and the test fake) rather than
// reaching for the concrete *voicetel.Client.
package sdkclient

import (
	"context"

	voicetel "github.com/voicetel/go-sdk"
)

// Client is the minimal surface the CLI relies on. A real *voicetel.Client
// satisfies it; tests substitute a fake.
type Client interface {
	// Connection state
	BaseURL() string
	APIKey() string

	// Auth
	Login(ctx context.Context, username int, password string) (string, error)
	SetAPIKey(key string)

	// Resource accessors — return the concrete service pointers so callers
	// can use the SDK's typed methods directly.
	Account() *voicetel.AccountService
	ACL() *voicetel.ACLService
	Authentication() *voicetel.AuthenticationService
	E911() *voicetel.E911Service
	Gateways() *voicetel.GatewaysService
	INumbering() *voicetel.INumberingService
	Lookups() *voicetel.LookupsService
	Messaging() *voicetel.MessagingService
	Numbers() *voicetel.NumbersService
	Support() *voicetel.SupportService
}

// realClient adapts *voicetel.Client to the Client interface. Because the
// SDK's APIKey field cannot be re-assigned after construction (it lives on
// an unexported transport), we hold the construction options and rebuild
// when SetAPIKey is called.
type realClient struct {
	inner   *voicetel.Client
	baseURL string
	ua      string
}

// New builds a real SDK-backed Client.
func New(baseURL, apiKey, userAgent string) Client {
	r := &realClient{baseURL: baseURL, ua: userAgent}
	r.rebuild(apiKey)
	return r
}

func (r *realClient) rebuild(apiKey string) {
	opts := []voicetel.Option{voicetel.WithUserAgent(r.ua)}
	if r.baseURL != "" {
		opts = append(opts, voicetel.WithBaseURL(r.baseURL))
	}
	if apiKey != "" {
		opts = append(opts, voicetel.WithAPIKey(apiKey))
	}
	r.inner = voicetel.NewClient(opts...)
}

func (r *realClient) BaseURL() string { return r.inner.BaseURL() }
func (r *realClient) APIKey() string  { return r.inner.APIKey() }

func (r *realClient) Login(ctx context.Context, username int, password string) (string, error) {
	// Login calls into the underlying transport, which mutates the
	// embedded apiKey. So we don't need to rebuild — the existing client
	// instance is good.
	return r.inner.Login(ctx, username, password)
}

func (r *realClient) SetAPIKey(key string) { r.rebuild(key) }

func (r *realClient) Account() *voicetel.AccountService               { return r.inner.Account }
func (r *realClient) ACL() *voicetel.ACLService                       { return r.inner.ACL }
func (r *realClient) Authentication() *voicetel.AuthenticationService { return r.inner.Authentication }
func (r *realClient) E911() *voicetel.E911Service                     { return r.inner.E911 }
func (r *realClient) Gateways() *voicetel.GatewaysService             { return r.inner.Gateways }
func (r *realClient) INumbering() *voicetel.INumberingService         { return r.inner.INumbering }
func (r *realClient) Lookups() *voicetel.LookupsService               { return r.inner.Lookups }
func (r *realClient) Messaging() *voicetel.MessagingService           { return r.inner.Messaging }
func (r *realClient) Numbers() *voicetel.NumbersService               { return r.inner.Numbers }
func (r *realClient) Support() *voicetel.SupportService               { return r.inner.Support }
