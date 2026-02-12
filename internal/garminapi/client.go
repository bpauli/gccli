package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// NewClient creates a new Garmin Connect API client.
func NewClient(tokens *garminauth.Tokens, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{},
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

// ConnectAPI makes a JSON API request to the Garmin Connect API.
// The response body is returned as a json.RawMessage. For 204 No Content
// responses (common for delete operations), nil is returned.
func (c *Client) ConnectAPI(ctx context.Context, method, path string, body io.Reader) (json.RawMessage, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
	}
	return c.doAPI(ctx, method, path, bodyBytes, true)
}

func (c *Client) doAPI(ctx context.Context, method, path string, body []byte, canRetry bool) (json.RawMessage, error) {
	reqURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.tokens.OAuth2AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", garminauth.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, resp.Body)
		if canRetry {
			if refreshErr := c.refreshToken(ctx); refreshErr == nil {
				return c.doAPI(ctx, method, path, body, false)
			}
		}
		return nil, &TokenExpiredError{Email: c.tokens.Email}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, &RateLimitError{RetryAfter: resp.Header.Get("Retry-After")}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &GarminAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(data),
		}
	}

	if resp.StatusCode == http.StatusNoContent || len(data) == 0 {
		return nil, nil
	}

	return json.RawMessage(data), nil
}

// Download fetches a binary resource from the API.
func (c *Client) Download(ctx context.Context, path string) ([]byte, error) {
	return c.doDownload(ctx, path, true)
}

func (c *Client) doDownload(ctx context.Context, path string, canRetry bool) ([]byte, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.tokens.OAuth2AccessToken)
	req.Header.Set("User-Agent", garminauth.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, resp.Body)
		if canRetry {
			if refreshErr := c.refreshToken(ctx); refreshErr == nil {
				return c.doDownload(ctx, path, false)
			}
		}
		return nil, &TokenExpiredError{Email: c.tokens.Email}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, &RateLimitError{RetryAfter: resp.Header.Get("Retry-After")}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &GarminAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(data),
		}
	}

	return data, nil
}

// refreshToken attempts to refresh the OAuth2 token using stored OAuth1 credentials.
func (c *Client) refreshToken(ctx context.Context) error {
	if !c.tokens.HasOAuth1() {
		return fmt.Errorf("no OAuth1 credentials for refresh")
	}

	newTokens, err := refreshTokensFn(ctx, c.tokens, garminauth.LoginOptions{
		Domain: c.tokens.Domain,
	})
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	c.tokens = newTokens
	return nil
}
