package conformance

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compactJSON(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		t.Fatalf("%v in %s", err, s)
	}
	return buf.String()
}

func scrubBody(t *testing.T, method, uri, body string) (Record, error) {
	t.Helper()
	return Scrub(Record{Method: method, Host: "connectapi.garmin.com", URI: uri, Status: 200, Body: json.RawMessage(body)})
}

func TestScrubReplacesEveryValue(t *testing.T) {
	for _, tc := range []struct{ name, uri, body, want string }{
		{
			"strings become the fixed word", "/userprofile-service/userprofile/settings",
			`{"email":"jane.doe@mail.example.org","displayName":"jane-runner","fullName":"Jane Doe","garminGUID":"8f14e45f-ceea-467f-a0e6-0a1b2c3d4e5f","timeZone":"Europe/Berlin","profileImageUrlLarge":"https://s3.example.com/profile/3141592.png","typeKey":"street_running","gender":"FEMALE","empty":""}`,
			`{"displayName":"tester","email":"tester","empty":"tester","fullName":"tester","garminGUID":"tester","gender":"tester","profileImageUrlLarge":"tester","timeZone":"tester","typeKey":"tester"}`,
		},
		{
			"digit strings stay digits", "/device-service/deviceregistration/devices",
			`[{"serialNumber":"3QZ123456","unitId":"3312345678","deviceId":"7"}]`,
			`[{"deviceId":"1","serialNumber":"tester","unitId":"1"}]`,
		},
		{
			"dates and times keep their format", "/activity-service/activity/21234567890",
			`{"birthDate":"1990-05-17","startTimeLocal":"2026-10-14 07:30:00","startTimeGMT":"2026-10-14T05:30:00.0","startTimeHhMm":"09:15","endGMT":"2026-10-14T05:30:00Z","offset":"2026-10-14T07:30:00+02:00"}`,
			`{"birthDate":"2026-01-01","endGMT":"2026-01-01T00:00:00Z","offset":"2026-01-01T00:00:00+00:00","startTimeGMT":"2026-01-01T00:00:00.0","startTimeHhMm":"00:00","startTimeLocal":"2026-01-01 00:00:00"}`,
		},
		{
			"numbers keep their class", "/userprofile-service/userprofile/settings",
			`{"weight":61000.0,"height":1.68e2,"steps":8123,"offset":-7200,"profileId":3141592,"startLatitude":52.5163}`,
			`{"height":0.5,"offset":1,"profileId":1,"startLatitude":0.5,"steps":1,"weight":0.5}`,
		},
		{
			"bools and nulls", "/userprofile-service/userprofile/user-settings",
			`{"userData":{"pregnant":true,"privacyProtected":false,"note":null}}`,
			`{"userData":{"note":null,"pregnant":true,"privacyProtected":true}}`,
		},
		{
			"nested arrays", "/web-gateway/course/owner/",
			`{"coursesForUser":[{"courseName":"Jane's loop","geoPoints":[{"latitude":52.5,"longitude":13.4},{"latitude":52.6,"longitude":13.5}]}],"heartRateValues":[[1760738400000,58],[1760738520000,null]]}`,
			`{"coursesForUser":[{"courseName":"tester","geoPoints":[{"latitude":0.5,"longitude":0.5},{"latitude":0.5,"longitude":0.5}]}],"heartRateValues":[[1,1],[1,null]]}`,
		},
		{
			"enum keys stay and date keys become the fixed date", "/userstats-service/wellness/daily/jane-runner",
			`{"allMetrics":{"metricsMap":{"WELLNESS_RESTING_HEART_RATE":[{"calendarDate":"2026-10-17","value":52.0}],"BODY_BATTERY":[]}},"byDate":{"2026-10-17":58,"2026-10-18":61}}`,
			`{"allMetrics":{"metricsMap":{"BODY_BATTERY":[],"WELLNESS_RESTING_HEART_RATE":[{"calendarDate":"2026-01-01","value":0.5}]}},"byDate":{"2026-01-01":1}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := scrubBody(t, "GET", tc.uri, tc.body)
			if err != nil {
				t.Fatalf("Scrub: %v", err)
			}
			if string(got.Body) != compactJSON(t, tc.want) {
				t.Errorf("Scrub body = %s\nwant %s", got.Body, compactJSON(t, tc.want))
			}
		})
	}
}

func TestScrubReplacesPathAndQuery(t *testing.T) {
	for _, tc := range []struct{ uri, want string }{
		{"/usersummary-service/usersummary/daily/jane-runner?calendarDate=2026-10-18", "/usersummary-service/usersummary/daily/tester?calendarDate=2026-01-01"},
		{"/activity-service/activity/21987654321/splits", "/activity-service/activity/1/splits"},
		{"/course-service/course/favorites", "/course-service/course/favorites"},
		{"/calendar-service/events?startDate=2026-10-18&pageIndex=1&limit=100&sortOrder=eventDate_asc", "/calendar-service/events?startDate=2026-01-01&pageIndex=1&limit=1&sortOrder=tester"},
		{"/course-service/course/elevation?smoothingEnabled=true", "/course-service/course/elevation?smoothingEnabled=tester"},
		{"/personalrecord-service/personalrecord/prs/jane.runner?email=jane%40runner.example", "/personalrecord-service/personalrecord/prs/tester?email=tester"},
	} {
		got, err := Scrub(Record{Method: "GET", Host: "connectapi.garmin.com", URI: tc.uri, Status: 200})
		if err != nil {
			t.Errorf("Scrub(%s): %v", tc.uri, err)
			continue
		}
		if got.URI != tc.want {
			t.Errorf("Scrub(%s).URI = %s, want %s", tc.uri, got.URI, tc.want)
		}
	}
}

func TestScrubFailsOnBadKey(t *testing.T) {
	_, err := scrubBody(t, "GET", "/userprofile-service/userprofile/settings",
		`{"displayName":"jane","userData":{"nickname":"Janey","Jane Runner":{"x":1},"pets":[{"petName":"Rex"},{"a b":1}]}}`)
	var bad *KeyError
	if !errors.As(err, &bad) {
		t.Fatalf("Scrub with a bad key: err = %v, want a KeyError", err)
	}
	msg := err.Error()
	for _, want := range []string{"$.<key 1>.<key 0>:", "$.<key 1>.<key 2>[1].<key 0>:", "GET /userprofile-service/userprofile/settings"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the error does not name %q:\n%s", want, msg)
		}
	}
	for _, text := range []string{"Jane", "jane", "Rex", "a b", "nickname", "userData", "pets", "displayName"} {
		if strings.Contains(msg, text) {
			t.Errorf("the error prints %q from the data:\n%s", text, msg)
		}
	}
}

func TestScrubFailsOnBadQueryKey(t *testing.T) {
	_, err := Scrub(Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/goal-service/goal/goals?status=active&Jane%20Runner=jane", Status: 200})
	if err == nil || !strings.Contains(err.Error(), "?<key 1>:") || strings.Contains(err.Error(), "Jane") || strings.Contains(err.Error(), "jane") || strings.Contains(err.Error(), "status") {
		t.Errorf("Scrub with a bad query key: err = %v, want one that names ?<key 1> and nothing from the query", err)
	}
}

func TestScrubFailsOnUnknownPath(t *testing.T) {
	_, err := Scrub(Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/usersummary-service/usersummary/weekly/jane-runner", Status: 200})
	if err == nil {
		t.Fatal("Scrub accepted a path that matches no template")
	}
	if want := "/usersummary-service/usersummary/*/*"; !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v, want it to show the path as %s", err, want)
	}
}

func TestScrubFailsOnOtherHost(t *testing.T) {
	_, err := Scrub(Record{Method: "GET", Host: "sso.garmin.com", URI: "/goal-service/goal/goals", Status: 200})
	if err == nil {
		t.Error("Scrub accepted a record from sso.garmin.com")
	}
}

func TestScrubIsIdempotent(t *testing.T) {
	first, err := scrubBody(t, "GET", "/web-gateway/course/owner/",
		`{"coursesForUser":[{"courseId":9,"courseName":"Loop","ownerDisplayName":"jane","startLatitude":52.5,"createdDate":"2026-09-01T10:00:00.0","favorite":false,"courseType":{"typeKey":"running"},"byDate":{"2026-09-01":1}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Scrub(first)
	if err != nil {
		t.Fatalf("Scrub of a scrubbed record: %v", err)
	}
	if string(second.Body) != string(first.Body) || second.URI != first.URI {
		t.Errorf("a second scrub changed the record:\n%s\n%s", first.Body, second.Body)
	}
}

func TestDecodeRecordRefusesUnknownFields(t *testing.T) {
	_, err := DecodeRecord([]byte(`{"method":"GET","host":"connectapi.garmin.com","uri":"/x","status":200,"headers":{"Authorization":["Bearer x"]}}`))
	if err == nil {
		t.Error("DecodeRecord accepted a record with headers")
	}
}

func writeRecords(t *testing.T, recs ...Record) string {
	t.Helper()
	dir := t.TempDir()
	for i, r := range recs {
		data, err := EncodeRecord(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, string(rune('a'+i))+".json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestScrubDirWritesEachRecordOnce(t *testing.T) {
	profile := func(name string) Record {
		return Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/userprofile-service/userprofile/settings", Status: 200, Body: json.RawMessage(`{"displayName":"` + name + `"}`)}
	}
	in := writeRecords(t, profile("jane"), profile("john"),
		Record{Method: "DELETE", Host: "connectapi.garmin.com", URI: "/activity-service/activity/42", Status: 204})
	out := filepath.Join(t.TempDir(), "out")
	if err := ScrubDir(in, out); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 || !strings.HasPrefix(names[0], "delete_activity-service.activity.1_204_") || !strings.HasPrefix(names[1], "get_userprofile-service.userprofile.settings_200_") {
		t.Errorf("ScrubDir wrote %v, want one DELETE and one profile record", names)
	}
}

func TestScrubDirWritesNothingOnFailure(t *testing.T) {
	in := writeRecords(t,
		Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/userprofile-service/userprofile/settings", Status: 200, Body: json.RawMessage(`{"displayName":"jane"}`)},
		Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/userprofile-service/userprofile/settings", Status: 200, Body: json.RawMessage(`{"userData":{"Jane Runner":1,"jane@runner.example":2}}`)},
		Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/goal-service/goal/goals", Status: 200, Body: json.RawMessage(`[{"go fast":1}]`)},
	)
	out := filepath.Join(t.TempDir(), "out")
	err := ScrubDir(in, out)
	if err == nil {
		t.Fatal("ScrubDir succeeded with bad keys")
	}
	for _, want := range []string{"$.<key 0>.<key 0>: ", "$.<key 0>.<key 1>: ", "$[0].<key 0>: ", "(1 records, first b.json)", "(1 records, first c.json)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not name %q:\n%v", want, err)
		}
	}
	for _, text := range []string{"Jane", "jane", "go fast", "userData"} {
		if strings.Contains(err.Error(), text) {
			t.Errorf("the error prints %q from the data:\n%v", text, err)
		}
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("ScrubDir created %s although the scrub failed", out)
	}
}

func recordedDir() string {
	return cmp.Or(os.Getenv("GCCLI_RECORDED_DIR"), "testdata/recorded")
}

type recordedFile struct {
	name string
	Record
}

func recordedFiles(t *testing.T, dir string) []recordedFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no record files in %s", dir)
	}
	var out []recordedFile
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		r, err := DecodeRecord(data)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		out = append(out, recordedFile{filepath.Base(path), r})
	}
	return out
}

func TestRecordedScrubbed(t *testing.T) {
	for _, f := range recordedFiles(t, recordedDir()) {
		got, err := Scrub(f.Record)
		if err != nil {
			t.Errorf("%s: %v", f.name, err)
			continue
		}
		want, err := os.ReadFile(filepath.Join(recordedDir(), f.name))
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeRecord(got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, want) {
			t.Errorf("%s is not scrubbed: the scrubber changes it to\n%s", f.name, data)
		}
	}
}
