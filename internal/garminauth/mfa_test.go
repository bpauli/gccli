package garminauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsMFARequired(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "MFA challenge page",
			html: `<html><head><title>GARMIN > MFA Challenge</title></head><body></body></html>`,
			want: true,
		},
		{
			name: "MFA in lowercase title",
			html: `<html><head><title>Garmin > MFA Verification</title></head><body></body></html>`,
			want: true,
		},
		{
			name: "success page no MFA",
			html: `<html><head><title>Success</title></head><body></body></html>`,
			want: false,
		},
		{
			name: "no title",
			html: `<html><body>no title</body></html>`,
			want: false,
		},
		{
			name: "empty html",
			html: "",
			want: false,
		},
		{
			name: "error page",
			html: `<html><head><title>GARMIN > Error</title></head><body></body></html>`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMFARequired(tt.html)
			if got != tt.want {
				t.Errorf("isMFARequired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPromptMFAFrom(t *testing.T) {
	t.Run("valid code", func(t *testing.T) {
		r := strings.NewReader("123456\n")
		var w strings.Builder
		code, err := promptMFAFrom(&w, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "123456" {
			t.Errorf("code = %q, want %q", code, "123456")
		}
		if !strings.Contains(w.String(), "Enter MFA code") {
			t.Error("expected prompt message")
		}
	})

	t.Run("code with whitespace", func(t *testing.T) {
		r := strings.NewReader("  789012  \n")
		var w strings.Builder
		code, err := promptMFAFrom(&w, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "789012" {
			t.Errorf("code = %q, want %q", code, "789012")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		r := strings.NewReader("\n")
		var w strings.Builder
		_, err := promptMFAFrom(&w, r)
		if err == nil {
			t.Fatal("expected error for empty input")
		}
		if !strings.Contains(err.Error(), "empty MFA code") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "empty MFA code")
		}
	})

	t.Run("EOF with no input", func(t *testing.T) {
		r := strings.NewReader("")
		var w strings.Builder
		_, err := promptMFAFrom(&w, r)
		if err == nil {
			t.Fatal("expected error for EOF")
		}
		if !strings.Contains(err.Error(), "no MFA code") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "no MFA code")
		}
	})
}

func TestResolveMFACode(t *testing.T) {
	t.Run("pre-supplied code", func(t *testing.T) {
		code, err := resolveMFACode(LoginOptions{MFACode: "123456"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "123456" {
			t.Errorf("code = %q, want %q", code, "123456")
		}
	})

	t.Run("pre-supplied takes priority over prompt", func(t *testing.T) {
		code, err := resolveMFACode(LoginOptions{
			MFACode: "111111",
			PromptMFA: func() (string, error) {
				return "222222", nil
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "111111" {
			t.Errorf("code = %q, want %q (pre-supplied should take priority)", code, "111111")
		}
	})

	t.Run("prompt function", func(t *testing.T) {
		code, err := resolveMFACode(LoginOptions{
			PromptMFA: func() (string, error) {
				return "654321", nil
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "654321" {
			t.Errorf("code = %q, want %q", code, "654321")
		}
	})

	t.Run("no code or prompt", func(t *testing.T) {
		_, err := resolveMFACode(LoginOptions{})
		if err == nil {
			t.Fatal("expected error when no code or prompt")
		}
		if !strings.Contains(err.Error(), "MFA required") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "MFA required")
		}
	})

	t.Run("prompt returns error", func(t *testing.T) {
		_, err := resolveMFACode(LoginOptions{
			PromptMFA: func() (string, error) {
				return "", &testError{msg: "prompt failed"}
			},
		})
		if err == nil {
			t.Fatal("expected error when prompt fails")
		}
		if !strings.Contains(err.Error(), "prompt failed") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "prompt failed")
		}
	})
}

func TestResolveMFACode_PromptError(t *testing.T) {
	_, err := resolveMFACode(LoginOptions{
		PromptMFA: func() (string, error) {
			return "", &testError{msg: "terminal closed"}
		},
	})
	if err == nil {
		t.Fatal("expected error when prompt fails")
	}
	if !strings.Contains(err.Error(), "terminal closed") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "terminal closed")
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// ssoMuxWithMFA returns an http.ServeMux that simulates the Garmin SSO flow
// with MFA challenge, requiring the given MFA code to succeed.
func ssoMuxWithMFA(t *testing.T, expectedMFACode string) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()

	// SSO embed (cookie setup).
	mux.HandleFunc("GET /sso/embed", func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "GARMIN-SSO-GUID", Value: "test-guid"})
		w.WriteHeader(http.StatusOK)
	})

	// SSO signin GET (return HTML with CSRF token).
	mux.HandleFunc("GET /sso/signin", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>GARMIN > Sign In</title></head>
<body><form><input type="hidden" name="_csrf" value="mock-csrf-token-123"></form></body></html>`))
	})

	// SSO signin POST — returns MFA challenge page.
	mux.HandleFunc("POST /sso/signin", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("username") != "test@example.com" || r.Form.Get("password") != "correctpass" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><title>GARMIN > Error</title></head><body>Invalid credentials</body></html>`))
			return
		}
		// Return MFA challenge page with CSRF token.
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>GARMIN > MFA Challenge</title></head>
<body><form action="/sso/verifyMFA/loginEnterMfaCode">
<input type="hidden" name="_csrf" value="mfa-csrf-token-789">
<input type="text" name="mfa-code">
</form></body></html>`))
	})

	// MFA verification endpoint.
	mux.HandleFunc("POST /sso/verifyMFA/loginEnterMfaCode", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("_csrf") != "mfa-csrf-token-789" {
			http.Error(w, "bad csrf", http.StatusForbidden)
			return
		}
		if r.Form.Get("mfa-code") != expectedMFACode {
			// Wrong MFA code — return MFA page again.
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><title>GARMIN > MFA Challenge</title></head>
<body><form><input type="hidden" name="_csrf" value="mfa-csrf-token-789">
<input type="text" name="mfa-code"></form></body></html>`))
			return
		}
		// Correct MFA code — return ticket.
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Success</title></head>
<body><script>window.location.replace("https://sso.garmin.com/sso/embed?ticket=ST-mfa-ticket-999")</script></body></html>`))
	})

	// OAuth consumer credentials.
	mux.HandleFunc("GET /oauth_consumer.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oauthConsumer{
			ConsumerKey:    "test-consumer-key",
			ConsumerSecret: "test-consumer-secret",
		})
	})

	// OAuth1 preauthorized exchange.
	mux.HandleFunc("GET /oauth-service/oauth/preauthorized", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "OAuth ") {
			http.Error(w, "missing OAuth header", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = w.Write([]byte("oauth_token=mock-oauth1-token&oauth_token_secret=mock-oauth1-secret&mfa_token=mock-mfa-token"))
	})

	// OAuth2 token exchange.
	mux.HandleFunc("POST /oauth-service/oauth/exchange/user/2.0", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "OAuth ") {
			http.Error(w, "missing OAuth header", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":               "Bearer",
			"access_token":             "mock-mfa-access-token",
			"refresh_token":            "mock-mfa-refresh-token",
			"expires_in":               3600,
			"refresh_token_expires_in": 7776000,
		})
	})

	return mux
}

func TestSubmitMFA(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("POST /sso/verifyMFA/loginEnterMfaCode", func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			if r.Form.Get("_csrf") != "csrf-abc" {
				http.Error(w, "bad csrf", http.StatusForbidden)
				return
			}
			if r.Form.Get("mfa-code") != "123456" {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<html><head><title>GARMIN > MFA Challenge</title></head></html>`))
				return
			}
			_, _ = w.Write([]byte(`<html><head><title>Success</title></head>
<body><script>window.location.replace("https://sso.garmin.com/sso/embed?ticket=ST-success-ticket")</script></body></html>`))
		})

		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		ep := Endpoints{
			SSOVerifyMFA: srv.URL + "/sso/verifyMFA/loginEnterMfaCode",
			SSOSignin:    srv.URL + "/sso/signin",
		}

		ticket, err := submitMFA(context.Background(), srv.Client(), ep, "csrf-abc", "123456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ticket != "ST-success-ticket" {
			t.Errorf("ticket = %q, want %q", ticket, "ST-success-ticket")
		}
	})

	t.Run("invalid MFA code", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("POST /sso/verifyMFA/loginEnterMfaCode", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><title>GARMIN > MFA Challenge</title></head></html>`))
		})

		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		ep := Endpoints{
			SSOVerifyMFA: srv.URL + "/sso/verifyMFA/loginEnterMfaCode",
			SSOSignin:    srv.URL + "/sso/signin",
		}

		_, err := submitMFA(context.Background(), srv.Client(), ep, "csrf", "wrong-code")
		if err == nil {
			t.Fatal("expected error for invalid MFA code")
		}
		if !strings.Contains(err.Error(), "invalid MFA code") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "invalid MFA code")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`<html><head><title>Server Error</title></head></html>`))
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{
			SSOVerifyMFA: srv.URL + "/sso/verifyMFA/loginEnterMfaCode",
			SSOSignin:    srv.URL + "/sso/signin",
		}

		_, err := submitMFA(context.Background(), srv.Client(), ep, "csrf", "123456")
		if err == nil {
			t.Fatal("expected error for server error response")
		}
	})
}

func TestLoginHeadless_MFAWithCode(t *testing.T) {
	mux := ssoMuxWithMFA(t, "123456")
	srv, ep := mockSSO(t, mux)

	origURL := oauthConsumerURL
	oauthConsumerURL = srv.URL + "/oauth_consumer.json"
	t.Cleanup(func() { oauthConsumerURL = origURL })

	tokens, err := loginHeadless(context.Background(), "test@example.com", "correctpass",
		LoginOptions{MFACode: "123456"}, ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokens.Email != "test@example.com" {
		t.Errorf("email = %q, want %q", tokens.Email, "test@example.com")
	}
	if tokens.OAuth2AccessToken != "mock-mfa-access-token" {
		t.Errorf("access_token = %q, want %q", tokens.OAuth2AccessToken, "mock-mfa-access-token")
	}
	if tokens.MFAToken != "mock-mfa-token" {
		t.Errorf("mfa_token = %q, want %q", tokens.MFAToken, "mock-mfa-token")
	}
	if tokens.IsExpired() {
		t.Error("token should not be expired")
	}
}

func TestLoginHeadless_MFAWithPrompt(t *testing.T) {
	mux := ssoMuxWithMFA(t, "654321")
	srv, ep := mockSSO(t, mux)

	origURL := oauthConsumerURL
	oauthConsumerURL = srv.URL + "/oauth_consumer.json"
	t.Cleanup(func() { oauthConsumerURL = origURL })

	promptCalled := false
	tokens, err := loginHeadless(context.Background(), "test@example.com", "correctpass",
		LoginOptions{
			PromptMFA: func() (string, error) {
				promptCalled = true
				return "654321", nil
			},
		}, ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !promptCalled {
		t.Error("PromptMFA was not called")
	}
	if tokens.OAuth2AccessToken != "mock-mfa-access-token" {
		t.Errorf("access_token = %q, want %q", tokens.OAuth2AccessToken, "mock-mfa-access-token")
	}
}

func TestLoginHeadless_MFANoCodeOrPrompt(t *testing.T) {
	mux := ssoMuxWithMFA(t, "123456")
	_, ep := mockSSO(t, mux)

	_, err := loginHeadless(context.Background(), "test@example.com", "correctpass",
		LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error when MFA required but no code or prompt")
	}
	if !strings.Contains(err.Error(), "MFA required") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "MFA required")
	}
}

func TestLoginHeadless_MFAWrongCode(t *testing.T) {
	mux := ssoMuxWithMFA(t, "123456")
	srv, ep := mockSSO(t, mux)

	origURL := oauthConsumerURL
	oauthConsumerURL = srv.URL + "/oauth_consumer.json"
	t.Cleanup(func() { oauthConsumerURL = origURL })

	_, err := loginHeadless(context.Background(), "test@example.com", "correctpass",
		LoginOptions{MFACode: "wrong-code"}, ep)
	if err == nil {
		t.Fatal("expected error for wrong MFA code")
	}
	if !strings.Contains(err.Error(), "invalid MFA code") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "invalid MFA code")
	}
}
