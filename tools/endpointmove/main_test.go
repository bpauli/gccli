package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

var before = map[string]string{
	"go.mod": "module example.com/m\n\ngo 1.26.0\n",
	"api/endpoint.go": `package api

var endpoints = mustIndex(
	otherEndpoints,
)

func mustIndex(...[]Endpoint) map[string]bool { return nil }
`,
	"api/passthrough_test.go": `package api

var passthroughMethods = []string{
	"GetDaily", "GetHealthPlans", "GetKeep", "GetPlainPath", "GetWeekly", "Other",
}
`,
	"api/health.go": `package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetDaily returns the daily summary.
func (c *Client) GetDaily(ctx context.Context, name, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/sum/daily/%s?from=%s&until=%s&metric=60", name, date, date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetWeekly returns weeks of data.
func (c *Client) GetWeekly(ctx context.Context, end string, weeks int) (json.RawMessage, error) {
	path := fmt.Sprintf("/sum/weekly/%s/%d", end, weeks)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetPlainPath has a concatenated path.
func (c *Client) GetPlainPath(ctx context.Context, date string) (json.RawMessage, error) {
	path := "/plain/" + date
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetHealthPlans takes a default locale.
func (c *Client) GetHealthPlans(ctx context.Context, locale string) (json.RawMessage, error) {
	if locale == "" {
		locale = "en"
	}
	return c.ConnectAPI(ctx, http.MethodGet, fmt.Sprintf("/plans?locale=%s", locale), nil)
}

// GetKeep picks one of two paths, so it stays a method.
func (c *Client) GetKeep(ctx context.Context, date string) (json.RawMessage, error) {
	if date == "" {
		return c.ConnectAPI(ctx, http.MethodGet, "/keep", nil)
	}
	return c.ConnectAPI(ctx, http.MethodGet, "/keep/"+date, nil)
}

// GetFloat prints a float, which a row cannot hold, so it stays a method.
func (c *Client) GetFloat(ctx context.Context, f float64) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, fmt.Sprintf("/f/%.1f", f), nil)
}

// GetHash has a # in its path, which a row cannot hold.
func (c *Client) GetHash(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/a#b", nil)
}

// Composite calls a moved method.
func (c *Client) Composite(ctx context.Context) error {
	_, err := c.GetPlainPath(ctx, "2024-01-01")
	return err
}
`,
	"cli/client.go": `package cli

import "example.com/m/api"

var client *api.Client
`,
	"cli/health.go": `package cli

import (
	"context"
	"fmt"
)

func run(ctx context.Context, weeks int) error {
	if _, err := client.GetWeekly(ctx, "2024-01-31", weeks); err != nil {
		return fmt.Errorf("weekly: %w", err)
	}
	_, err := client.GetHealthPlans(ctx,
		"de")
	return err
}
`,
	"other/same.go": `package other

func (x *T) GetWeekly(a, b string, n int) {}
`,
}

var after = map[string]string{
	"api/endpoint.go": `package api

var endpoints = mustIndex(
	otherEndpoints,
	healthEndpoints,
)

func mustIndex(...[]Endpoint) map[string]bool { return nil }
`,
	"api/passthrough_test.go": `package api

var passthroughMethods = []string{
	"GetKeep", "Other",
}
`,
	"api/health.go": `package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var healthEndpoints = []Endpoint{
	{Name: "health.daily", Method: http.MethodGet, Host: ConnectAPI, Path: "/sum/daily/{name}?from={date}&until={date}&metric=60"},
	{Name: "health.weekly", Method: http.MethodGet, Host: ConnectAPI, Path: "/sum/weekly/{end}/{weeks}"},
	{Name: "health.plain-path", Method: http.MethodGet, Host: ConnectAPI, Path: "/plain/{date}"},
	{
		Name: "health.plans", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/plans?locale={locale}",
		Params: []Param{{Name: "locale", Default: "en"}},
	},
}

// GetKeep picks one of two paths, so it stays a method.
func (c *Client) GetKeep(ctx context.Context, date string) (json.RawMessage, error) {
	if date == "" {
		return c.ConnectAPI(ctx, http.MethodGet, "/keep", nil)
	}
	return c.ConnectAPI(ctx, http.MethodGet, "/keep/"+date, nil)
}

// GetFloat prints a float, which a row cannot hold, so it stays a method.
func (c *Client) GetFloat(ctx context.Context, f float64) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, fmt.Sprintf("/f/%.1f", f), nil)
}

// GetHash has a # in its path, which a row cannot hold.
func (c *Client) GetHash(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/a#b", nil)
}

// Composite calls a moved method.
func (c *Client) Composite(ctx context.Context) error {
	_, err := c.Call(ctx, "health.plain-path", map[string]string{"date": "2024-01-01"}, nil)
	return err
}
`,
	"cli/health.go": `package cli

import (
	"context"
	"fmt"
	"strconv"
)

func run(ctx context.Context, weeks int) error {
	if _, err := client.Call(ctx, "health.weekly", map[string]string{"end": "2024-01-31", "weeks": strconv.Itoa(weeks)}, nil); err != nil {
		return fmt.Errorf("weekly: %w", err)
	}
	_, err := client.Call(ctx, "health.plans", map[string]string{"locale": "de"}, nil)
	return err
}
`,
}

func TestMoveRewritesTableCallersAndList(t *testing.T) {
	root := t.TempDir()
	for name, src := range before {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	changed, err := move(root, "api/health.go", "")
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	var want []string
	for name := range after {
		want = append(want, filepath.Join(root, name))
	}
	slices.Sort(want)
	if !slices.Equal(changed, want) {
		t.Errorf("changed %q\nwant %q", changed, want)
	}
	for name, src := range before {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		exp, ok := after[name]
		if !ok {
			exp = src
		}
		if string(got) != exp {
			t.Errorf("%s:\n%s\nwant:\n%s", name, got, exp)
		}
	}

	again, err := move(root, "api/health.go", "")
	if err != nil || len(again) != 0 {
		t.Errorf("second run changed %q, err %v", again, err)
	}
}

func TestMoveAppendsToAnExistingTable(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                  before["go.mod"],
		"api/endpoint.go":         after["api/endpoint.go"],
		"api/passthrough_test.go": "package api\n\nvar passthroughMethods = []string{\n\t\"GetLate\", \"Other\",\n}\n",
		"api/health.go": `package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var healthEndpoints = []Endpoint{
	{Name: "health.daily", Method: http.MethodGet, Host: ConnectAPI, Path: "/sum/daily"},
}

func (c *Client) GetLate(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/late", nil)
}
`,
	}
	for name, src := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := move(root, "api/health.go", ""); err != nil {
		t.Fatalf("move: %v", err)
	}
	want := map[string]string{
		"api/endpoint.go":         after["api/endpoint.go"],
		"api/passthrough_test.go": "package api\n\nvar passthroughMethods = []string{\n\t\"Other\",\n}\n",
		"api/health.go": `package api

import "net/http"

var healthEndpoints = []Endpoint{
	{Name: "health.daily", Method: http.MethodGet, Host: ConnectAPI, Path: "/sum/daily"},
	{Name: "health.late", Method: http.MethodGet, Host: ConnectAPI, Path: "/late"},
}
`,
	}
	for name, exp := range want {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != exp {
			t.Errorf("%s:\n%s\nwant:\n%s", name, got, exp)
		}
	}
}

func TestMoveHandlesBodiesAndOptionalPairs(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                  before["go.mod"],
		"api/endpoint.go":         after["api/endpoint.go"],
		"api/passthrough_test.go": "package api\n\nvar passthroughMethods = []string{\n\t\"GetThings\", \"Other\", \"UpdateThing\",\n}\n",
		"api/things.go": `package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetThings has an optional filter.
func (c *Client) GetThings(ctx context.Context, start int, kind, end string) (json.RawMessage, error) {
	path := fmt.Sprintf("/things?start=%d", start)
	if kind != "" {
		path += "&kind=" + kind
	}
	if end != "" {
		path += "&end=" + end
	}
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// UpdateThing sends the payload as it is.
func (c *Client) UpdateThing(ctx context.Context, id string, payload json.RawMessage) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPut, "/things/"+id, bytes.NewReader(payload))
}

// GetTwice uses its filter in the path too, so it stays a method.
func (c *Client) GetTwice(ctx context.Context, kind string) (json.RawMessage, error) {
	path := "/twice/" + kind
	if kind != "" {
		path += "?kind=" + kind
	}
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// SendBuilt sends a body it builds, so it stays a method.
func (c *Client) SendBuilt(ctx context.Context, id string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/built/"+id, bytes.NewReader([]byte(id)))
}
`,
		"cli/client.go": before["cli/client.go"],
		"cli/things.go": `package cli

import "context"

func run(ctx context.Context, kind string, body []byte) error {
	if _, err := client.GetThings(ctx, 0, kind, ""); err != nil {
		return err
	}
	_, err := client.UpdateThing(ctx, "7", body)
	return err
}
`,
	}
	for name, src := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := move(root, "api/things.go", ""); err != nil {
		t.Fatalf("move: %v", err)
	}
	want := map[string]string{
		"api/passthrough_test.go": "package api\n\nvar passthroughMethods = []string{\n\t\"Other\",\n}\n",
		"api/things.go": `package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

var thingsEndpoints = []Endpoint{
	{
		Name: "things.list", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/things?start={start}&kind={kind}&end={end}",
		Params: []Param{{Name: "kind", Optional: true}, {Name: "end", Optional: true}},
	},
	{Name: "things.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/things/{id}", Body: true},
}

// GetTwice uses its filter in the path too, so it stays a method.
func (c *Client) GetTwice(ctx context.Context, kind string) (json.RawMessage, error) {
	path := "/twice/" + kind
	if kind != "" {
		path += "?kind=" + kind
	}
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// SendBuilt sends a body it builds, so it stays a method.
func (c *Client) SendBuilt(ctx context.Context, id string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/built/"+id, bytes.NewReader([]byte(id)))
}
`,
		"cli/things.go": `package cli

import "context"

func run(ctx context.Context, kind string, body []byte) error {
	if _, err := client.Call(ctx, "things.list", map[string]string{"start": "0", "kind": kind, "end": ""}, nil); err != nil {
		return err
	}
	_, err := client.Call(ctx, "things.update", map[string]string{"id": "7"}, body)
	return err
}
`,
	}
	for name, exp := range want {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != exp {
			t.Errorf("%s:\n%s\nwant:\n%s", name, got, exp)
		}
	}
	if again, err := move(root, "api/things.go", ""); err != nil || len(again) != 0 {
		t.Errorf("second run changed %q, err %v", again, err)
	}
}

func TestAction(t *testing.T) {
	tests := []struct{ method, area, want string }{
		{"GetDailySummary", "health", "daily-summary"},
		{"GetSPO2", "health", "spo2"},
		{"GetCyclingFTP", "health", "cycling-ftp"},
		{"GetHRV", "health", "hrv"},
		{"GetTrainingPlans", "training", "plans"},
		{"GetTrainingReadiness", "health", "training-readiness"},
		{"RequestReload", "wellness", "request-reload"},
		{"GetHealth", "health", "get"},
		{"GetActivities", "activities", "list"},
		{"GetActivity", "activities", "get"},
		{"GetActivityDetails", "activities", "details"},
		{"DeleteActivity", "activities", "delete"},
		{"SearchActivities", "activities", "search"},
		{"GetCourseFavorites", "courses", "favorites"},
		{"UnscheduleWorkout", "workouts", "unschedule"},
	}
	for _, tc := range tests {
		if got := action(tc.method, tc.area); got != tc.want {
			t.Errorf("action(%q, %q) = %q, want %q", tc.method, tc.area, got, tc.want)
		}
	}
}
