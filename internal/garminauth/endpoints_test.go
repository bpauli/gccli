package garminauth

import (
	"net/url"
	"testing"
)

func TestNewEndpoints_Global(t *testing.T) {
	t.Parallel()

	ep := NewEndpoints(DomainGlobal)

	if ep.SSOBase != "https://sso.garmin.com" {
		t.Errorf("SSOBase = %q, want %q", ep.SSOBase, "https://sso.garmin.com")
	}
	if ep.SSOEmbed != "https://sso.garmin.com/sso/embed" {
		t.Errorf("SSOEmbed = %q, want %q", ep.SSOEmbed, "https://sso.garmin.com/sso/embed")
	}
	if ep.SSOSignin != "https://sso.garmin.com/sso/signin" {
		t.Errorf("SSOSignin = %q, want %q", ep.SSOSignin, "https://sso.garmin.com/sso/signin")
	}
	if ep.SSOVerifyMFA != "https://sso.garmin.com/sso/verifyMFA/loginEnterMfaCode" {
		t.Errorf("SSOVerifyMFA = %q, want %q", ep.SSOVerifyMFA, "https://sso.garmin.com/sso/verifyMFA/loginEnterMfaCode")
	}
	if ep.OAuthBase != "https://connectapi.garmin.com" {
		t.Errorf("OAuthBase = %q, want %q", ep.OAuthBase, "https://connectapi.garmin.com")
	}
	if ep.ConnectAPI != "https://connectapi.garmin.com" {
		t.Errorf("ConnectAPI = %q, want %q", ep.ConnectAPI, "https://connectapi.garmin.com")
	}
}

func TestNewEndpoints_China(t *testing.T) {
	t.Parallel()

	ep := NewEndpoints(DomainChina)

	if ep.SSOBase != "https://sso.garmin.cn" {
		t.Errorf("SSOBase = %q, want %q", ep.SSOBase, "https://sso.garmin.cn")
	}
	if ep.SSOEmbed != "https://sso.garmin.cn/sso/embed" {
		t.Errorf("SSOEmbed = %q, want %q", ep.SSOEmbed, "https://sso.garmin.cn/sso/embed")
	}
	if ep.ConnectAPI != "https://connectapi.garmin.cn" {
		t.Errorf("ConnectAPI = %q, want %q", ep.ConnectAPI, "https://connectapi.garmin.cn")
	}
}

func TestEndpoints_PreauthorizedURL(t *testing.T) {
	t.Parallel()

	ep := NewEndpoints(DomainGlobal)
	ticket := "ST-123456-abc"
	loginURL := "https://sso.garmin.com/sso/embed"

	got := ep.PreauthorizedURL(ticket, loginURL)

	// Verify the URL contains expected components.
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("PreauthorizedURL returned invalid URL: %v", err)
	}
	if u.Host != "connectapi.garmin.com" {
		t.Errorf("host = %q, want %q", u.Host, "connectapi.garmin.com")
	}
	if u.Path != "/oauth-service/oauth/preauthorized" {
		t.Errorf("path = %q, want %q", u.Path, "/oauth-service/oauth/preauthorized")
	}
	q := u.Query()
	if q.Get("ticket") != ticket {
		t.Errorf("ticket = %q, want %q", q.Get("ticket"), ticket)
	}
	if q.Get("login-url") != loginURL {
		t.Errorf("login-url = %q, want %q", q.Get("login-url"), loginURL)
	}
	if q.Get("accepts-mfa-tokens") != "true" {
		t.Errorf("accepts-mfa-tokens = %q, want %q", q.Get("accepts-mfa-tokens"), "true")
	}
}

func TestEndpoints_ExchangeURL(t *testing.T) {
	t.Parallel()

	ep := NewEndpoints(DomainGlobal)

	got := ep.ExchangeURL()
	want := "https://connectapi.garmin.com/oauth-service/oauth/exchange/user/2.0"
	if got != want {
		t.Errorf("ExchangeURL() = %q, want %q", got, want)
	}
}

func TestEndpoints_ExchangeURL_China(t *testing.T) {
	t.Parallel()

	ep := NewEndpoints(DomainChina)

	got := ep.ExchangeURL()
	want := "https://connectapi.garmin.cn/oauth-service/oauth/exchange/user/2.0"
	if got != want {
		t.Errorf("ExchangeURL() = %q, want %q", got, want)
	}
}

func TestDomainConstants(t *testing.T) {
	t.Parallel()

	if DomainGlobal != "garmin.com" {
		t.Errorf("DomainGlobal = %q, want %q", DomainGlobal, "garmin.com")
	}
	if DomainChina != "garmin.cn" {
		t.Errorf("DomainChina = %q, want %q", DomainChina, "garmin.cn")
	}
}
