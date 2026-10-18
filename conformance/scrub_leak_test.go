package conformance

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScrubLeavesNoPlantedString(t *testing.T) {
	for _, tc := range []struct {
		name, uri, body string
		planted         []string
	}{
		{
			"name and email as map keys", "/userstats-service/wellness/daily/jane-runner",
			`{"allMetrics":{"metricsMap":{"Jane Runner":1,"jane@runner.example":true}}}`,
			[]string{"Jane Runner", "jane@runner.example", "jane-runner"},
		},
		{
			"sleep qualifier and hrv status", "/wellness-service/wellness/dailySleepData/jane-runner",
			`{"dailySleepDTO":{"sleepScores":{"overall":{"qualifierKey":"POOR","value":41}}},"hrvStatus":"UNBALANCED"}`,
			[]string{"POOR", "UNBALANCED", "41", "jane-runner"},
		},
		{
			"body battery and feedback", "/usersummary-service/usersummary/daily/jane-runner",
			`{"bodyBatteryDynamicFeedbackEvent":{"bodyBatteryLevel":"LOW","feedbackShortType":"SLEEP_PREPARATION_NOISY","feedbackLongType":"SLEEP_PREPARATION_NOISY_LONG"}}`,
			[]string{"LOW", "SLEEP_PREPARATION_NOISY", "jane-runner"},
		},
		{
			"activity type and training effect", "/activity-service/activity/21234567890",
			`{"activityTypeDTO":{"typeKey":"yoga"},"summaryDTO":{"trainingEffectLabel":"RECOVERY"}}`,
			[]string{"yoga", "RECOVERY", "21234567890"},
		},
		{
			"free text under enum keys", "/calendar-service/events",
			`[{"eventPrivacy":{"label":"Burkhard"},"sportType":{"sportTypeKey":"burkhard_pauli"},"goalCategory":"Berlin","goalStatus":"mail-bpauli-de","userLevel":"Jane-Runner"}]`,
			[]string{"Burkhard", "burkhard_pauli", "Berlin", "mail-bpauli-de", "Jane-Runner"},
		},
		{
			"secrets under key names", "/userprofile-service/userprofile/settings",
			`{"consumerKey":"fc3e99d2-118c-44b8-8ae3-03370dde24c0","accessKey":"AKIAJANERUNNER","appVersion":"Burkhard.Pauli"}`,
			[]string{"fc3e99d2", "AKIAJANERUNNER", "Burkhard"},
		},
		{
			"device and exercise names", "/device-service/deviceregistration/devices",
			`[{"productDisplayName":"Burkhard Pauli","partNumber":"006-B4315-00","exerciseSets":[{"exercises":[{"name":"BurkhardsLift"}]}]}]`,
			[]string{"Burkhard Pauli", "006-B4315-00", "BurkhardsLift"},
		},
		{
			"unknown query key", "/goal-service/goal/goals?Jane%20Runner=jane",
			`[]`,
			[]string{"Jane", "jane"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Scrub(Record{Method: "GET", Host: "connectapi.garmin.com", URI: tc.uri, Status: 200, Body: json.RawMessage(tc.body)})
			out := got.URI + string(got.Body)
			if err != nil {
				out = err.Error()
			}
			for _, p := range tc.planted {
				if strings.Contains(out, p) {
					t.Errorf("%q is in the scrub result:\n%s", p, out)
				}
			}
		})
	}
}

func TestScrubOutputDoesNotDependOnScalars(t *testing.T) {
	scrub := func(body string) string {
		t.Helper()
		got, err := Scrub(Record{Method: "GET", Host: "connectapi.garmin.com", URI: "/userprofile-service/userprofile/user-settings", Status: 200, Body: json.RawMessage(body)})
		if err != nil {
			t.Fatal(err)
		}
		return string(got.Body)
	}
	a := scrub(`{"userData":{"pregnant":true,"weight":61000.5,"steps":8123,"hrvStatus":"POOR","birthDate":"1990-05-17","note":null,"flags":[true,false]}}`)
	b := scrub(`{"userData":{"pregnant":false,"weight":70000.25,"steps":17,"hrvStatus":"BALANCED","birthDate":"2001-12-31","note":null,"flags":[false,true]}}`)
	if a != b {
		t.Errorf("the scrub output depends on the input values:\n%s\n%s", a, b)
	}
}
