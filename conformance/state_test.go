package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const diTokenPath = "/di-oauth2-service/oauth/token"

func stateSandbox(t *testing.T, g *garmin, tokenFile string) *sandbox {
	t.Helper()
	token, err := os.ReadFile(filepath.Join("testdata/state", tokenFile))
	if err != nil {
		t.Fatal(err)
	}
	s := newSandbox(t, binary(t), g)
	s.importToken(t, token)
	return s
}

func (s *sandbox) gccli(t *testing.T, args ...string) result {
	t.Helper()
	res := s.run(t, "", append([]string{"--account", "test@example.com"}, args...)...)
	if res.code != 0 {
		t.Fatalf("gccli %s: exit %d\nstderr: %s", strings.Join(args, " "), res.code, res.stderr)
	}
	return res
}

func refreshes(reqs []request) []request {
	var out []request
	for _, r := range reqs {
		if r.URI == diTokenPath {
			out = append(out, r)
		}
	}
	return out
}

func TestRefreshRace(t *testing.T) {
	t.Parallel()
	s := stateSandbox(t, newGarmin(500*time.Millisecond, "AT-old"), "token_expired.json")

	args := []string{"--account", "test@example.com", "--json", "devices", "list"}
	waits := []func() result{s.start(t, "", args...), s.start(t, "", args...)}
	for i, wait := range waits {
		if res := wait(); res.code != 0 {
			t.Errorf("process %d: exit %d\nstderr: %s", i+1, res.code, res.stderr)
		}
	}

	reqs := s.proxy.requests()
	if got := refreshes(reqs); len(got) != 1 {
		t.Errorf("the token service got %d refresh requests, want 1: both processes refreshed the same token, so the token lock did not hold\nrequests: %v", len(got), reqs)
	}
	for _, r := range reqs {
		if r.URI != diTokenPath && r.Bearer != "AT-1" {
			t.Errorf("%s used the bearer token %q, want the refreshed AT-1", r, r.Bearer)
		}
	}
}

func TestAuthTokenRefresh(t *testing.T) {
	t.Parallel()
	s := stateSandbox(t, newGarmin(0), "token_expired.json")

	for i := range 2 {
		if res := s.gccli(t, "auth", "token"); res.stdout != "AT-1\n" {
			t.Errorf("auth token run %d printed %q, want the refreshed token AT-1", i+1, res.stdout)
		}
	}
	if got := refreshes(s.proxy.requests()); len(got) != 1 {
		t.Errorf("the token service got %d refresh requests, want 1: auth token did not save the refreshed token\nrequests: %v", len(got), got)
	}
}

func TestDomainChinaRefresh(t *testing.T) {
	t.Parallel()
	s := stateSandbox(t, newGarmin(0, "AT-old"), "token_expired_cn.json")

	s.gccli(t, "--json", "devices", "list")
	var status struct{ Domain string }
	if err := json.Unmarshal([]byte(s.gccli(t, "--json", "auth", "status").stdout), &status); err != nil {
		t.Fatal(err)
	}
	if status.Domain != "garmin.cn" {
		t.Errorf("auth status after the refresh shows domain %q, want garmin.cn", status.Domain)
	}

	reqs := s.proxy.requests()
	want := map[string]string{diTokenPath: "diauth.garmin.cn", "/device-service/deviceregistration/devices": "connectapi.garmin.cn"}
	for _, r := range reqs {
		if !strings.HasSuffix(r.Host, ".garmin.cn") {
			t.Errorf("a garmin.cn token sent %s to a host outside garmin.cn", r)
		}
		if host, ok := want[r.URI]; ok {
			if r.Host != host {
				t.Errorf("%s went to %s, want %s", r, r.Host, host)
			}
			delete(want, r.URI)
		}
	}
	for uri, host := range want {
		t.Errorf("no request for %s reached %s\nrequests: %v", uri, host, reqs)
	}
}

func TestDomainFromConfig(t *testing.T) {
	t.Parallel()
	s := stateSandbox(t, newGarmin(0), "token_nodomain.json")
	writeConfig(t, s, `{"default_account": "test@example.com", "domain": "garmin.cn"}`)

	s.gccli(t, "--json", "devices", "list")
	reqs := s.proxy.requests()
	if len(reqs) == 0 {
		t.Fatal("devices list sent no request")
	}
	for _, r := range reqs {
		if r.Host != "connectapi.garmin.cn" {
			t.Errorf("with domain garmin.cn in config.json, %s went to %s, want connectapi.garmin.cn", r, r.Host)
		}
	}
}

func TestConfigWriteKeepsKeys(t *testing.T) {
	t.Parallel()
	s := stateSandbox(t, newGarmin(0), "token_golden.json")
	seed, err := os.ReadFile("testdata/state/config_future_key.json")
	if err != nil {
		t.Fatal(err)
	}
	writeConfig(t, s, string(seed))

	s.importToken(t, []byte(`{"oauth2_access_token":"AT","oauth2_expires_at":"2099-01-01T00:00:00Z","domain":"garmin.com","email":"other@example.com"}`))
	data, err := os.ReadFile(filepath.Join(s.home, "gccli", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("config.json after auth import is not JSON: %v\n%s", err, data)
	}
	if err := json.Unmarshal(seed, &want); err != nil {
		t.Fatal(err)
	}
	want["default_account"] = "other@example.com"
	if !reflect.DeepEqual(got, want) {
		t.Errorf("config.json after auth import:\n%s\nwant the seed keys kept and default_account set to other@example.com", data)
	}
}

func writeConfig(t *testing.T, s *sandbox, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.home, "gccli", "config.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
