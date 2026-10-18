package conformance

import (
	"bytes"
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

func TestScrubReplacesPersonalFields(t *testing.T) {
	for _, tc := range []struct{ name, uri, body, want string }{
		{
			"email", "/userprofile-service/userprofile/settings",
			`{"email":"jane.doe@mail.example.org","userName":"jane.doe@mail.example.org","contactEmail":"x"}`,
			`{"contactEmail":"t@example.com","email":"t@example.com","userName":"t@example.com"}`,
		},
		{
			"display name", "/userprofile-service/userprofile/settings",
			`{"displayName":"jane-runner","ownerDisplayName":"8f14e45f-ceea-467f"}`,
			`{"displayName":"tester","ownerDisplayName":"tester"}`,
		},
		{
			"names", "/userprofile-service/userprofile/settings",
			`{"fullName":"Jane Doe","firstName":"Jane","lastName":"Doe","ownerFullName":"Jane Doe"}`,
			`{"firstName":"Test User","fullName":"Test User","lastName":"Test User","ownerFullName":"Test User"}`,
		},
		{
			"profile ids", "/userprofile-service/userprofile/settings",
			`{"profileId":3141592,"userProfilePK":3141592,"garminGUID":"8f14e45f-ceea-467f-a0e6-0a1b2c3d4e5f","id":"3141592"}`,
			`{"garminGUID":"test-id","id":"test-id","profileId":1,"userProfilePK":1}`,
		},
		{
			"device serials", "/device-service/deviceregistration/devices",
			`[{"serialNumber":"3QZ123456","unitId":3312345678,"deviceId":"3312345678"}]`,
			`[{"deviceId":"test-id","serialNumber":"TEST-SERIAL","unitId":1}]`,
		},
		{
			"GPS coordinates", "/activity-service/activity/21234567890",
			`{"startLatitude":52.5163,"startLongitude":13.3777,"summaryDTO":{"endLatitude":-33.86,"endLongitude":151.2}}`,
			`{"startLatitude":0.5,"startLongitude":0.5,"summaryDTO":{"endLatitude":0.5,"endLongitude":0.5}}`,
		},
		{
			"tokens", "/userprofile-service/userprofile/settings",
			`{"accessToken":"eyJhbGciOi","refresh_token":"RT-secret","sessionId":"abc"}`,
			`{"accessToken":"test-token","refresh_token":"test-token","sessionId":"test-token"}`,
		},
		{
			"free text and places", "/activity-service/activity/21234567890",
			`{"activityName":"Run to Anna's house","description":"Meet at Alexanderplatz","locationName":"Berlin"}`,
			`{"activityName":"Test","description":"Test","locationName":"Test"}`,
		},
		{
			"urls and time zones", "/userprofile-service/userprofile/settings",
			`{"profileImageUrlLarge":"https://s3.example.com/profile/3141592.png","timeZone":"Europe/Berlin","unitKey":"Europe/Berlin"}`,
			`{"profileImageUrlLarge":"https://example.com/test","timeZone":"UTC","unitKey":"UTC"}`,
		},
		{
			"dates and times", "/activity-service/activity/21234567890",
			`{"birthDate":"1990-05-17","startTimeLocal":"2026-10-14 07:30:00","startTimeGMT":"2026-10-14T05:30:00.0","startTimeHhMm":"09:15"}`,
			`{"birthDate":"2026-01-01","startTimeGMT":"2026-01-01T00:00:00.0","startTimeHhMm":"00:00","startTimeLocal":"2026-01-01 00:00:00"}`,
		},
		{
			"nested arrays", "/web-gateway/course/owner/",
			`{"coursesForUser":[{"courseName":"Jane's loop","geoPoints":[{"latitude":52.5,"longitude":13.4},{"latitude":52.6,"longitude":13.5}]}],"heartRateValues":[[1760738400000,58],[1760738520000,null]]}`,
			`{"coursesForUser":[{"courseName":"Test","geoPoints":[{"latitude":0.5,"longitude":0.5},{"latitude":0.5,"longitude":0.5}]}],"heartRateValues":[[1,1],[1,null]]}`,
		},
		{
			"enums stay", "/activity-service/activity/21234567890",
			`{"activityType":{"typeKey":"street_running"},"manufacturer":"GARMIN","measurementSystem":"statute_us","race":true,"note":null,"exerciseSets":[{"setType":"ACTIVE","exercises":[{"category":"BENCH_PRESS","name":"BARBELL_BENCH_PRESS"}]}]}`,
			`{"activityType":{"typeKey":"street_running"},"exerciseSets":[{"exercises":[{"category":"BENCH_PRESS","name":"BARBELL_BENCH_PRESS"}],"setType":"ACTIVE"}],"manufacturer":"GARMIN","measurementSystem":"statute_us","note":null,"race":true}`,
		},
		{
			"numbers keep their class", "/userprofile-service/userprofile/settings",
			`{"weight":61000.0,"height":1.68e2,"steps":8123,"offset":-7200}`,
			`{"height":0.5,"offset":1,"steps":1,"weight":0.5}`,
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
		{"/calendar-service/events?startDate=2026-10-18&pageIndex=1&limit=100&sortOrder=eventDate_asc", "/calendar-service/events?startDate=2026-01-01&pageIndex=1&limit=1&sortOrder=eventDate_asc"},
		{"/course-service/course/elevation?smoothingEnabled=true", "/course-service/course/elevation?smoothingEnabled=true"},
		{"/goal-service/goal/goals?status=active", "/goal-service/goal/goals?status=active"},
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

func TestScrubFailsOnUnknownString(t *testing.T) {
	_, err := scrubBody(t, "GET", "/userprofile-service/userprofile/settings",
		`{"displayName":"jane","userData":{"nickname":"Janey","pets":[{"petName":"Rex"}]}}`)
	var unclassified *UnclassifiedError
	if !errors.As(err, &unclassified) {
		t.Fatalf("Scrub of an unknown string: err = %v, want an UnclassifiedError", err)
	}
	msg := err.Error()
	for _, want := range []string{"userData.nickname", "userData.pets[].petName", "GET /userprofile-service/userprofile/settings"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the error does not name %q:\n%s", want, msg)
		}
	}
	for _, value := range []string{"Janey", "Rex", "jane"} {
		if strings.Contains(msg, value) {
			t.Errorf("the error prints the value %q:\n%s", value, msg)
		}
	}
}

func TestScrubFailsOnUnknownQueryKey(t *testing.T) {
	_, err := Scrub(Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/goal-service/goal/goals?owner=jane", Status: 200})
	if err == nil || !strings.Contains(err.Error(), "?owner") || strings.Contains(err.Error(), "jane") {
		t.Errorf("Scrub of an unknown query key: err = %v, want one that names ?owner and not its value", err)
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
		`{"coursesForUser":[{"courseId":9,"courseName":"Loop","ownerDisplayName":"jane","startLatitude":52.5,"createdDate":"2026-09-01T10:00:00.0","courseType":{"typeKey":"running"}}]}`)
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
		Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/userprofile-service/userprofile/settings", Status: 200, Body: json.RawMessage(`{"nickname":"Janey","pets":["Rex"]}`)},
		Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/goal-service/goal/goals", Status: 200, Body: json.RawMessage(`[{"motto":"go"}]`)},
	)
	out := filepath.Join(t.TempDir(), "out")
	err := ScrubDir(in, out)
	if err == nil {
		t.Fatal("ScrubDir succeeded with unknown strings")
	}
	for _, want := range []string{"nickname", "pets[]", "[].motto"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not name %s:\n%v", want, err)
		}
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("ScrubDir created %s although the scrub failed", out)
	}
}
