package garminauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

// defaultBrowserTimeout is the default time to wait for the browser callback
// if the caller's context has no deadline.
const defaultBrowserTimeout = 2 * time.Minute

// openBrowserFn is the function used to open a URL in the user's browser.
// It is a variable to allow overriding in tests.
var openBrowserFn = openBrowser

// LoginBrowser performs SSO login by opening the Garmin SSO page in the user's
// default browser. After the user authenticates, Garmin redirects to a local
// callback server where the service ticket is captured and exchanged for OAuth tokens.
func LoginBrowser(ctx context.Context, email string, opts LoginOptions) (*Tokens, error) {
	ep := NewEndpoints(opts.domain())
	return loginBrowser(ctx, email, opts, ep)
}

// loginBrowser is the internal implementation that accepts explicit endpoints
// for testability.
func loginBrowser(ctx context.Context, email string, opts LoginOptions, ep Endpoints) (*Tokens, error) {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	// Apply default timeout if context has no deadline.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultBrowserTimeout)
		defer cancel()
	}

	// Start local callback server.
	callbackURL, ticketCh, cleanup, err := startCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}
	defer cleanup()

	// Build SSO login URL with redirect to our local callback.
	ssoURL := buildSSOURL(ep, callbackURL, email)

	// Open browser to SSO login page.
	if err := openBrowserFn(ssoURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	// Wait for callback with service ticket.
	var ticket string
	select {
	case result := <-ticketCh:
		if result.err != nil {
			return nil, fmt.Errorf("callback: %w", result.err)
		}
		ticket = result.ticket
	case <-ctx.Done():
		return nil, fmt.Errorf("browser login timed out: %w", ctx.Err())
	}

	// Exchange ticket for tokens (reuse exchange logic from headless flow).
	consumer, err := fetchOAuthConsumer(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("fetch oauth consumer: %w", err)
	}

	oauth1Token, oauth1Secret, mfaToken, err := exchangePreauthorized(
		ctx, client, ep, consumer, ticket, callbackURL)
	if err != nil {
		return nil, fmt.Errorf("oauth1 exchange: %w", err)
	}

	tokens, err := exchangeOAuth2(
		ctx, client, ep, consumer, oauth1Token, oauth1Secret, mfaToken)
	if err != nil {
		return nil, fmt.Errorf("oauth2 exchange: %w", err)
	}

	tokens.Domain = ep.domain()
	tokens.Email = email
	tokens.OAuth1Token = oauth1Token
	tokens.OAuth1Secret = oauth1Secret
	if mfaToken != "" {
		tokens.MFAToken = mfaToken
	}

	return tokens, nil
}

// callbackResult holds the result of a browser SSO callback.
type callbackResult struct {
	ticket string
	err    error
}

// startCallbackServer starts a local HTTP server on a random port that
// listens for the SSO callback containing the service ticket.
func startCallbackServer() (callbackURL string, ticketCh <-chan callbackResult, cleanup func(), err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, nil, fmt.Errorf("listen: %w", err)
	}

	ch := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ticket := r.URL.Query().Get("ticket")
		if ticket == "" {
			// Ignore requests without a ticket (e.g. favicon, health checks).
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><h1>Authentication successful!</h1>` +
			`<p>You can close this window and return to the terminal.</p></body></html>`))

		// Send ticket to channel; drop if already received.
		select {
		case ch <- callbackResult{ticket: ticket}:
		default:
		}
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	cleanup = func() {
		_ = server.Close()
	}

	return callbackURL, ch, cleanup, nil
}

// buildSSOURL constructs the Garmin SSO login URL that redirects to the
// callback URL after successful authentication. If email is non-empty, it is
// passed as prepopUsername so the SSO form pre-fills the email field.
func buildSSOURL(ep Endpoints, callbackURL, email string) string {
	params := url.Values{
		"service":                         {callbackURL},
		"gauthHost":                       {ep.SSOBase + "/sso"},
		"source":                          {callbackURL},
		"redirectAfterAccountLoginUrl":    {callbackURL},
		"redirectAfterAccountCreationUrl": {callbackURL},
	}
	if email != "" {
		params.Set("prepopUsername", email)
	}
	return ep.SSOSignin + "?" + params.Encode()
}

// openBrowser opens the given URL in the user's default browser.
func openBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start()
	case "linux":
		return exec.Command("xdg-open", rawURL).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
