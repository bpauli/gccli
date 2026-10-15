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

// Client interacts with the Garmin Connect API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	tokens     *garminauth.Tokens

	// OnTokenRefresh is called after a successful OAuth2 token refresh.
	// Use this to persist refreshed tokens back to storage.
	// If nil, no action is taken (tokens are only updated in-memory).
	OnTokenRefresh func(tokens *garminauth.Tokens)
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

var stallTimeout = 60 * time.Second

func newHTTPClient() *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	return &http.Client{Transport: NewRetryTransport(&stallTransport{base: base, idle: stallTimeout})}
}

// NewClient creates a new Garmin Connect API client.
func NewClient(tokens *garminauth.Tokens, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: newHTTPClient(),
		baseURL:    garminauth.NewEndpoints(tokens.Domain).ConnectAPI,
		tokens:     tokens,
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
			if canRefresh && c.refreshToken(ctx) == nil {
				continue
			}
			return response{}, &TokenExpiredError{Email: c.tokens.Email}
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
	if resp.status == http.StatusNoContent || len(resp.body) == 0 {
		return nil, nil
	}
	return json.RawMessage(resp.body), nil
}

// Download fetches a binary resource from the API.
func (c *Client) Download(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.do(ctx, request{kind: kindDownload, method: http.MethodGet, path: path})
	if err != nil {
		return nil, err
	}
	return resp.body, nil
}

// refreshToken attempts to refresh the OAuth2 token using stored refresh credentials.
func (c *Client) refreshToken(ctx context.Context) error {
	if !c.tokens.CanRefresh() {
		return fmt.Errorf("no refresh credentials available")
	}

	newTokens, err := refreshTokensFn(ctx, c.tokens, garminauth.LoginOptions{
		Domain: c.tokens.Domain,
	})
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	c.tokens = newTokens
	if c.OnTokenRefresh != nil {
		c.OnTokenRefresh(newTokens)
	}
	return nil
}
