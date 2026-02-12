package garminauth

import "net/http"

// LoginOptions configures the authentication flow.
type LoginOptions struct {
	// Domain is the Garmin domain ("garmin.com" or "garmin.cn").
	// Defaults to DomainGlobal if empty.
	Domain string

	// MFACode is a pre-supplied MFA code for non-interactive login.
	// If empty and MFA is required, PromptMFA is called.
	MFACode string

	// PromptMFA is called to interactively prompt the user for an MFA code.
	// It should return the code entered by the user.
	// If nil and MFA is required without a pre-supplied code, login fails.
	PromptMFA func() (string, error)

	// HTTPClient is an optional HTTP client to use for SSO requests.
	// If nil, a default client with cookie jar is created.
	HTTPClient *http.Client
}

// domain returns the configured domain, defaulting to DomainGlobal.
func (o *LoginOptions) domain() string {
	if o.Domain != "" {
		return o.Domain
	}
	return DomainGlobal
}
