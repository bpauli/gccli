package cmd

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"

	"golang.org/x/term"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
)

// Function variables for dependency injection in tests.
var (
	loginBrowserFn  = garminauth.LoginBrowser
	loginHeadlessFn = garminauth.LoginHeadless
	readPasswordFn  = func(fd int) ([]byte, error) { return term.ReadPassword(fd) }
)

// AuthLoginCmd authenticates with Garmin Connect.
type AuthLoginCmd struct {
	Email    string `arg:"" help:"Garmin account email address."`
	Headless bool   `help:"Use headless login (email/password) instead of browser SSO."`
	MFACode  string `help:"MFA code for two-factor authentication." name:"mfa-code"`
}

func (c *AuthLoginCmd) Validate() error {
	_, err := loginDomain()
	return err
}

func loginDomain() (string, error) {
	cfg, err := config.Read()
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}
	return garminauth.ParseDomain(cfg.Domain())
}

func (c *AuthLoginCmd) Run(g *Globals) error {
	domain, err := loginDomain()
	if err != nil {
		return err
	}

	// Build an HTTP client with retry transport so auth requests
	// automatically retry on 429 (rate limit) and 5xx responses.
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("create cookie jar: %w", err)
	}
	retry := garminapi.NewRetryTransport(nil)
	retry.Retry5xxAnyMethod = true
	httpClient := &http.Client{
		Jar:       jar,
		Transport: retry,
	}

	opts := garminauth.LoginOptions{
		Domain:     domain,
		MFACode:    c.MFACode,
		HTTPClient: httpClient,
	}

	var tokens *garminauth.Tokens

	if c.Headless {
		g.UI.Infof("Logging in as %s (headless)...", c.Email)
		_, _ = fmt.Fprint(g.Stderr, "Password: ")
		pw, pwErr := readPasswordFn(int(os.Stdin.Fd()))
		_, _ = fmt.Fprintln(g.Stderr)
		if pwErr != nil {
			return fmt.Errorf("read password: %w", pwErr)
		}

		opts.PromptMFA = garminauth.PromptMFA
		tokens, err = loginHeadlessFn(g.Context, c.Email, string(pw), opts)
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}
	} else {
		g.UI.Infof("Opening browser for Garmin SSO login...")
		tokens, err = loginBrowserFn(g.Context, c.Email, opts)
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}
	}

	// Store tokens in keyring.
	store, err := loadSecretsStore()
	if err != nil {
		return err
	}

	data, err := tokens.Marshal()
	if err != nil {
		return fmt.Errorf("marshal tokens: %w", err)
	}

	if err := store.Set(c.Email, data); err != nil {
		return fmt.Errorf("store tokens: %w", err)
	}

	// Save as default account so subsequent commands don't need --account.
	if err := config.Update(func(f *config.File) { f.DefaultAccount = c.Email }); err != nil {
		return fmt.Errorf("save default account: %w", err)
	}

	g.UI.Successf("Logged in as %s (domain: %s)", c.Email, tokens.Domain)
	return nil
}
