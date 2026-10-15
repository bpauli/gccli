package garminauth

import "fmt"

// Supported Garmin domains.
const (
	DomainGlobal = "garmin.com"
	DomainChina  = "garmin.cn"
)

// SSO and OAuth endpoint path fragments.
const (
	ssoEmbedPath      = "/sso/embed"
	ssoSigninPath     = "/sso/signin"
	ssoVerifyMFAPath  = "/sso/verifyMFA/loginEnterMfaCode"
	oauthExchangePath = "/oauth-service/oauth/exchange/user/2.0"
	diTokenPath       = "/di-oauth2-service/oauth/token"
)

// OAuth consumer credentials URL (public, hosted by Garmin/Garth).
const OAuthConsumerURL = "https://thegarth.s3.amazonaws.com/oauth_consumer.json"

// User-Agent used for OAuth exchange requests (matches Garmin mobile app).
const UserAgent = "com.garmin.android.apps.connectmobile"

// Endpoints holds the resolved URLs for a given Garmin domain.
type Endpoints struct {
	// Domain is the Garmin domain the URLs were built from.
	Domain string
	// SSOBase is the base URL for SSO operations (e.g. "https://sso.garmin.com").
	SSOBase string
	// SSOEmbed is the SSO embed URL used for cookie setup.
	SSOEmbed string
	// SSOSignin is the SSO sign-in page URL.
	SSOSignin string
	// SSOVerifyMFA is the MFA verification endpoint.
	SSOVerifyMFA string
	// OAuthBase is the base URL for OAuth operations (e.g. "https://connectapi.garmin.com").
	OAuthBase string
	// ConnectAPI is the base URL for the Connect API (e.g. "https://connectapi.garmin.com").
	ConnectAPI string
	// DIToken is the DI OAuth2 token URL for ticket exchange and refresh.
	DIToken string
}

// NewEndpoints builds the full endpoint URLs for the given domain.
// Supported domains are "garmin.com" (global) and "garmin.cn" (China).
func NewEndpoints(domain string) Endpoints {
	ssoBase := fmt.Sprintf("https://sso.%s", domain)
	oauthBase := fmt.Sprintf("https://connectapi.%s", domain)

	return Endpoints{
		Domain:       domain,
		SSOBase:      ssoBase,
		SSOEmbed:     ssoBase + ssoEmbedPath,
		SSOSignin:    ssoBase + ssoSigninPath,
		SSOVerifyMFA: ssoBase + ssoVerifyMFAPath,
		OAuthBase:    oauthBase,
		ConnectAPI:   oauthBase,
		DIToken:      fmt.Sprintf("https://diauth.%s%s", domain, diTokenPath),
	}
}

// ExchangeURL returns the full OAuth1-to-OAuth2 token exchange URL.
func (e Endpoints) ExchangeURL() string {
	return e.OAuthBase + oauthExchangePath
}
