package garminauth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// isMFARequired checks whether the SSO response HTML indicates an MFA challenge.
func isMFARequired(htmlBody string) bool {
	title, _ := getTitle(htmlBody)
	return strings.Contains(strings.ToUpper(title), "MFA")
}

// PromptMFA reads an MFA code from the terminal via stdin.
// It prompts the user on stderr and reads a single line from stdin.
func PromptMFA() (string, error) {
	return promptMFAFrom(os.Stderr, os.Stdin)
}

// promptMFAFrom reads an MFA code using the given writer for prompts and reader for input.
// Extracted for testability.
func promptMFAFrom(w io.Writer, r io.Reader) (string, error) {
	_, _ = fmt.Fprint(w, "Enter MFA code: ")
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read MFA code: %w", err)
		}
		return "", fmt.Errorf("no MFA code provided")
	}
	code := strings.TrimSpace(scanner.Text())
	if code == "" {
		return "", fmt.Errorf("empty MFA code")
	}
	return code, nil
}

// submitMFA posts the MFA code to the Garmin SSO MFA verification endpoint
// and extracts the service ticket from the response.
// signinParams are the same query parameters used for the signin flow and must
// be forwarded to the MFA endpoint as query parameters (matching garth behaviour).
func submitMFA(ctx context.Context, client *http.Client, ep Endpoints, signinParams url.Values, csrf, mfaCode string) (string, error) {
	formData := url.Values{
		"mfa-code": {mfaCode},
		"embed":    {"true"},
		"_csrf":    {csrf},
		"fromPage": {"setupEnterMfaCode"},
	}

	mfaURL := ep.SSOVerifyMFA + "?" + signinParams.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		mfaURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("create MFA request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", ep.SSOSignin)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("submit MFA: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("read MFA response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limited by Garmin SSO (429) — wait a few minutes and try again")
	}

	// Check for MFA error (still on MFA page means code was wrong).
	if isMFARequired(string(body)) {
		return "", fmt.Errorf("invalid MFA code")
	}

	// After following redirects, the ticket may be in the final URL.
	if ticket := ticketFromURL(resp.Request.URL.String()); ticket != "" {
		return ticket, nil
	}

	// Extract service ticket from the response body (JS redirect pattern).
	ticket, err := getTicket(string(body))
	if err != nil {
		title, _ := getTitle(string(body))
		return "", fmt.Errorf("MFA verification failed (title=%q): %w", title, err)
	}

	return ticket, nil
}

// ticketFromURL extracts a service ticket from a URL string, handling the case
// where Garmin redirects to e.g. .../sso/embed?ticket=ST-... after MFA success.
func ticketFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("ticket")
}

// resolveMFACode returns the MFA code from LoginOptions.
// It uses the pre-supplied MFACode if set, otherwise calls PromptMFA.
// Returns an error if neither is available.
func resolveMFACode(opts LoginOptions) (string, error) {
	if opts.MFACode != "" {
		return opts.MFACode, nil
	}
	if opts.PromptMFA != nil {
		return opts.PromptMFA()
	}
	return "", fmt.Errorf("MFA required but no code provided and no prompt function configured; use --mfa-code or run interactively")
}
