package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
)

// refreshTokensFn is the function used to refresh OAuth2 tokens.
// Variable for dependency injection in tests.
var refreshTokensFn = garminauth.RefreshOAuth2

// Refresher returns the tokens to use after the API rejected sent with a 401.
// It returns a *TokenExpiredError when the tokens cannot be refreshed. The
// client returns any error from it unchanged.
type Refresher func(ctx context.Context, sent *garminauth.Tokens) (*garminauth.Tokens, error)

// Client interacts with the Garmin Connect API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	tokens     *garminauth.Tokens

	Refresher Refresher
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(cl *Client) { cl.httpClient = c }
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(u string) ClientOption {
	return func(cl *Client) { cl.baseURL = u }
}

// WithAttemptTransport wraps the transport of each attempt with wrap. The
// result sits under the retry transport and over the stall watch, so the
// wrapper sees every attempt and a stalled request still fails. The e2e
// recorder uses it.
func WithAttemptTransport(wrap func(http.RoundTripper) http.RoundTripper) ClientOption {
	return func(cl *Client) { cl.httpClient = newHTTPClient(wrap) }
}

var stallTimeout = 60 * time.Second

func newHTTPClient(wrap func(http.RoundTripper) http.RoundTripper) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	var attempt http.RoundTripper = &stallTransport{base: base, idle: stallTimeout}
	if wrap != nil {
		attempt = wrap(attempt)
	}
	return &http.Client{Transport: NewRetryTransport(attempt)}
}

// NewClient creates a new Garmin Connect API client.
func NewClient(tokens *garminauth.Tokens, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: newHTTPClient(nil),
		baseURL:    garminauth.NewEndpoints(tokens.Domain).ConnectAPI,
		tokens:     tokens,
		Refresher:  refreshInMemory,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Tokens returns the current tokens held by the client.
func (c *Client) Tokens() *garminauth.Tokens {
	return c.tokens
}

type kind int

const (
	kindJSON kind = iota
	kindDownload
	kindUpload
)

var kinds = [...]struct {
	accept, createErr, readErr string
	sendErr                    func(method, path string) string
}{
	kindJSON: {
		accept: "application/json", createErr: "create request", readErr: "read response",
		sendErr: func(method, path string) string { return "request " + method + " " + path },
	},
	kindDownload: {
		createErr: "create request", readErr: "read download response",
		sendErr: func(_, path string) string { return "download " + path },
	},
	kindUpload: {
		createErr: "create upload request", readErr: "read upload response",
		sendErr: func(_, path string) string { return "upload " + path },
	},
}

type request struct {
	kind        kind
	method      string
	path        string
	body        []byte
	contentType string
	noRefresh   bool
}

type response struct {
	status int
	body   []byte
}

func (c *Client) do(ctx context.Context, r request) (response, error) {
	k := kinds[r.kind]
	for canRefresh := !r.noRefresh; ; canRefresh = false {
		var body io.Reader
		if r.body != nil {
			body = bytes.NewReader(r.body)
		}
		req, err := http.NewRequestWithContext(ctx, r.method, c.baseURL+r.path, body)
		if err != nil {
			return response{}, fmt.Errorf("%s: %w", k.createErr, err)
		}
		req.Header.Set("Authorization", "Bearer "+c.tokens.OAuth2AccessToken)
		req.Header.Set("User-Agent", garminauth.UserAgent)
		if k.accept != "" {
			req.Header.Set("Accept", k.accept)
		}
		if r.contentType != "" {
			req.Header.Set("Content-Type", r.contentType)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return response{}, fmt.Errorf("%s: %w", k.sendErr(r.method, r.path), err)
		}
		data, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			if !canRefresh || !c.tokens.CanRefresh() {
				return response{}, &TokenExpiredError{Email: c.tokens.Email}
			}
			tokens, err := c.Refresher(ctx, c.tokens)
			if err != nil {
				return response{}, err
			}
			c.tokens = tokens
			continue
		case resp.StatusCode == http.StatusTooManyRequests:
			return response{}, &RateLimitError{RetryAfter: resp.Header.Get("Retry-After")}
		case err != nil:
			return response{}, fmt.Errorf("%s: %w", k.readErr, err)
		case resp.StatusCode >= 400:
			return response{}, &GarminAPIError{StatusCode: resp.StatusCode, Message: string(data)}
		}
		return response{status: resp.StatusCode, body: data}, nil
	}
}

// ConnectAPI makes a JSON API request to the Garmin Connect API.
// The response body is returned as a json.RawMessage. For 204 No Content
// responses (common for delete operations), nil is returned.
func (c *Client) ConnectAPI(ctx context.Context, method, path string, body io.Reader) (json.RawMessage, error) {
	r := request{kind: kindJSON, method: method, path: path}
	if body != nil {
		var err error
		r.body, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		r.contentType = "application/json"
	}
	resp, err := c.do(ctx, r)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp), nil
}

// Download fetches a binary resource from the API.
func (c *Client) Download(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.do(ctx, request{kind: kindDownload, method: http.MethodGet, path: path})
	if err != nil {
		return nil, err
	}
	return resp.body, nil
}

func refreshInMemory(ctx context.Context, sent *garminauth.Tokens) (*garminauth.Tokens, error) {
	tokens, err := refreshTokensFn(ctx, sent, garminauth.LoginOptions{Domain: sent.Domain})
	if err != nil {
		return nil, &TokenExpiredError{Email: sent.Email, Err: err}
	}
	return tokens, nil
}
