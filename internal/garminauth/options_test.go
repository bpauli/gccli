package garminauth

import "testing"

func TestLoginOptions_Domain_Default(t *testing.T) {
	t.Parallel()

	opts := &LoginOptions{}
	if got := opts.domain(); got != DomainGlobal {
		t.Errorf("domain() = %q, want %q", got, DomainGlobal)
	}
}

func TestLoginOptions_Domain_Custom(t *testing.T) {
	t.Parallel()

	opts := &LoginOptions{Domain: DomainChina}
	if got := opts.domain(); got != DomainChina {
		t.Errorf("domain() = %q, want %q", got, DomainChina)
	}
}

func TestLoginOptions_ZeroValue(t *testing.T) {
	t.Parallel()

	opts := &LoginOptions{}

	if opts.MFACode != "" {
		t.Errorf("MFACode = %q, want empty", opts.MFACode)
	}
	if opts.PromptMFA != nil {
		t.Error("PromptMFA should be nil by default")
	}
	if opts.HTTPClient != nil {
		t.Error("HTTPClient should be nil by default")
	}
}
