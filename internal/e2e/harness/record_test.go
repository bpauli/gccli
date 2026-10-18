package harness

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bpauli/gccli/conformance"
)

type answer struct {
	status int
	body   string
}

func answers(list ...answer) http.RoundTripper {
	return roundTripFunc(func(req *http.Request) *http.Response {
		a := list[0]
		list = list[1:]
		h := http.Header{}
		h.Set("Set-Cookie", "SESSIONID=cookie-secret")
		return &http.Response{StatusCode: a.status, Header: h, Body: io.NopCloser(strings.NewReader(a.body)), Request: req}
	})
}

func TestRecorderWritesOneFilePerResponse(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	rec, err := NewRecorder(dir)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: rec.Wrap(answers(
		answer{200, `{"displayName": "jane"}`},
		answer{200, "\x0e\x10FIT"},
		answer{204, ""},
	))}
	for _, uri := range []string{"/userprofile-service/userprofile/settings", "/workout-service/workout/FIT/1?x=1", "/activity-service/activity/1"} {
		req, err := http.NewRequest(http.MethodPost, "https://connectapi.garmin.com"+uri, strings.NewReader(`{"request":"body-secret"}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer token-secret")
		req.Header.Set("Cookie", "GARMIN-SSO=cookie-secret")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	want := map[string]conformance.Record{
		"0001.json": {Method: "POST", Host: "connectapi.garmin.com", URI: "/userprofile-service/userprofile/settings", Status: 200, Body: []byte(`{"displayName":"jane"}`)},
		"0002.json": {Method: "POST", Host: "connectapi.garmin.com", URI: "/workout-service/workout/FIT/1?x=1", Status: 200, NonJSONBytes: 5},
		"0003.json": {Method: "POST", Host: "connectapi.garmin.com", URI: "/activity-service/activity/1", Status: 204},
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		t.Fatalf("the recorder wrote %d files, want %d", len(entries), len(want))
	}
	for name, w := range want {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"Bearer", "token-secret", "cookie-secret", "body-secret", "Cookie", "Authorization"} {
			if strings.Contains(string(data), secret) {
				t.Errorf("%s holds %q: the recorder wrote a header or the request body\n%s", name, secret, data)
			}
		}
		got, err := conformance.DecodeRecord(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.Body != nil {
			var buf bytes.Buffer
			if err := json.Compact(&buf, got.Body); err != nil {
				t.Fatal(err)
			}
			got.Body = buf.Bytes()
		}
		if !reflect.DeepEqual(got, w) {
			t.Errorf("%s = %+v, want %+v", name, got, w)
		}
	}
}

func TestRecorderGivesTheClientTheBody(t *testing.T) {
	rec, err := NewRecorder(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Transport: rec.Wrap(answers(answer{200, `{"a": [1, 2]}`}))}).Get("https://connectapi.garmin.com/x")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"a": [1, 2]}` {
		t.Errorf("the client got %q, want the response body", body)
	}
}

func TestNewRecorderRefusesAFullDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRecorder(dir); err == nil {
		t.Error("NewRecorder accepted a directory that holds a record from an earlier run")
	}
}
