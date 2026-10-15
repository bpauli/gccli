package harness

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
)

func TestDecodeToken(t *testing.T) {
	want := &garminauth.Tokens{
		OAuth2AccessToken:  "AT",
		OAuth2RefreshToken: "RT",
		OAuth2ExpiresAt:    time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		DIClientID:         "DI",
		Domain:             garminauth.DomainGlobal,
		Email:              "e@example.com",
	}
	data, err := want.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeToken(base64.StdEncoding.EncodeToString(data))
	if err != nil {
		t.Fatalf("DecodeToken: %v", err)
	}
	if *got != *want {
		t.Errorf("DecodeToken = %+v, want %+v", got, want)
	}
}

func TestDecodeTokenRejects(t *testing.T) {
	for name, raw := range map[string]string{
		"not base64":      "notbase64",
		"not json":        base64.StdEncoding.EncodeToString([]byte("not json")),
		"no access token": base64.StdEncoding.EncodeToString([]byte(`{"oauth2_refresh_token":"RT"}`)),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeToken(raw)
			if err == nil || !strings.HasPrefix(err.Error(), "GARMIN_TOKEN ") {
				t.Errorf("err = %v, want an error that names GARMIN_TOKEN", err)
			}
		})
	}
}
