package cmd

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/testutil/requestlog"
)

type writeCase struct {
	args   []string
	files  map[string]string
	routes map[string]string
	result string
	prompt string

	json     string
	notes    string
	human    string
	fallback string
	table    string
	plain    string
}

const noContent = ""

const okOut = "{\n  \"ok\": true\n}\n"

var writeCases = map[string]writeCase{
	"activity upload": {
		args:   []string{"run.fit"},
		files:  map[string]string{"run.fit": "fit"},
		routes: map[string]string{"POST /upload-service/upload": `{"detailedImportResult":{"uploadId":5}}`},
		json:   "{\n  \"detailedImportResult\": {\n    \"uploadId\": 5\n  }\n}\n",
		human:  "Uploaded $DIR/run.fit\n",
	},
	"activity create": {
		args:     []string{"--name", "Swim", "--type", "swimming", "--duration", "30m", "--date", "2024-06-15T07:30:00"},
		routes:   map[string]string{"POST /activity-service/activity": `{"activityId":21234567890}`},
		json:     "{\n  \"activityId\": 21234567890\n}\n",
		human:    "Created activity 21234567890\n",
		fallback: "Activity created\n",
	},
	"activity rename": {
		args:   []string{"1", "Morning Run"},
		routes: map[string]string{"PUT /activity-service/activity/1": `{"activityId":1,"activityName":"Morning Run"}`},
		json:   "{\n  \"activityId\": 1,\n  \"activityName\": \"Morning Run\"\n}\n",
		human:  "Renamed activity 1 to \"Morning Run\"\n",
	},
	"activity retype": {
		args:   []string{"1", "--type-id", "1", "--type-key", "running"},
		routes: map[string]string{"PUT /activity-service/activity/1": `{"activityId":1}`},
		json:   "{\n  \"activityId\": 1\n}\n",
		human:  "Changed activity 1 type to running\n",
	},
	"activity describe": {
		args:     []string{"1", "Easy run"},
		routes:   map[string]string{"PUT /activity-service/activity/1": noContent},
		json:     okOut,
		human:    "Changed activity 1 description to \"Easy run\"\n",
		fallback: "Changed activity 1 description to \"Easy run\"\n",
	},
	"activity privacy": {
		args:     []string{"1", "connections"},
		routes:   map[string]string{"PUT /activity-service/activity/1": noContent},
		json:     okOut,
		human:    "Changed activity 1 privacy to connections\n",
		fallback: "Changed activity 1 privacy to connections\n",
	},
	"activity favorite": {
		args:     []string{"1"},
		routes:   map[string]string{"PUT /activity-service/activity/1": noContent},
		json:     okOut,
		human:    "Added activity 1 to favorites\n",
		fallback: "Added activity 1 to favorites\n",
	},
	"activity unfavorite": {
		args:     []string{"1"},
		routes:   map[string]string{"PUT /activity-service/activity/1": noContent},
		json:     okOut,
		human:    "Removed activity 1 from favorites\n",
		fallback: "Removed activity 1 from favorites\n",
	},
	"activity likes remove": {
		args: []string{"1"},
		routes: map[string]string{
			"GET /conversation-service/conversation/ACTIVITY/1?likerLimit=3": `{"conversationUuid":"c-1","likedByUser":true}`,
			"DELETE /conversation-service/conversation/like/c-1/0":           noContent,
		},
		result:   "DELETE /conversation-service/conversation/like/c-1/0",
		json:     okOut,
		human:    "Removed like from activity 1\n",
		fallback: "Removed like from activity 1\n",
	},
	"activity comments add": {
		args:     []string{"1", "Nice run"},
		routes:   map[string]string{"POST /conversation-service/conversation/comment/ACTIVITY/1/null": `{"conversationCommentUuid":"k-1"}`},
		json:     "{\n  \"conversationCommentUuid\": \"k-1\"\n}\n",
		human:    "Added comment k-1 to activity 1\n",
		fallback: "Added comment to activity 1\n",
	},
	"activity comments edit": {
		args: []string{"1", "k-1", "Great run"},
		routes: map[string]string{
			"GET /conversation-service/conversation/ACTIVITY/1?likerLimit=3": `{"conversationUuid":"c-1"}`,
			"PUT /conversation-service/conversation/comment/c-1/k-1":         noContent,
		},
		result:   "PUT /conversation-service/conversation/comment/c-1/k-1",
		json:     okOut,
		human:    "Changed comment k-1 on activity 1\n",
		fallback: "Changed comment k-1 on activity 1\n",
	},
	"activity comments delete": {
		args: []string{"1", "k-1"},
		routes: map[string]string{
			"GET /conversation-service/conversation/ACTIVITY/1?likerLimit=3": `{"conversationUuid":"c-1"}`,
			"DELETE /conversation-service/conversation/comment/c-1/k-1/0":    noContent,
		},
		result:   "DELETE /conversation-service/conversation/comment/c-1/k-1/0",
		prompt:   "Delete comment k-1 on activity 1?",
		json:     okOut,
		human:    "Deleted comment k-1 on activity 1\n",
		fallback: "Deleted comment k-1 on activity 1\n",
	},
	"activity comments likes remove": {
		args:     []string{"k-1"},
		routes:   map[string]string{"DELETE /conversation-service/conversation/comment/like/k-1/0": noContent},
		json:     okOut,
		human:    "Removed like from comment k-1\n",
		fallback: "Removed like from comment k-1\n",
	},
	"activity exercise-sets set": {
		args:   []string{"1", "-e", "BENCH_PRESS/BARBELL_BENCH_PRESS:12@20"},
		routes: map[string]string{"PUT /activity-service/activity/1/exerciseSets": noContent},
		json:   okOut,
		human:  "Exercise sets updated for activity 1\n",
	},
	"body add-weight": {
		args:   []string{"80.5"},
		routes: map[string]string{"POST /weight-service/user-weight": `{"samplePk":3}`},
		json:   "{\n  \"samplePk\": 3\n}\n",
		human:  "Added weight: 80.5 kg\n",
	},
	"body add-composition": {
		args:   []string{"80"},
		routes: map[string]string{"POST /upload-service/upload": `{"detailedImportResult":{"uploadId":6}}`},
		json:   "{\n  \"detailedImportResult\": {\n    \"uploadId\": 6\n  }\n}\n",
		human:  "Added body composition: 80.0 kg\n",
	},
	"body add-blood-pressure": {
		args:   []string{"--systolic", "120", "--diastolic", "80", "--pulse", "60"},
		routes: map[string]string{"POST /bloodpressure-service/bloodpressure": `{"version":4}`},
		json:   "{\n  \"version\": 4\n}\n",
		human:  "Added blood pressure: 120/80 mmHg, pulse 60 bpm\n",
	},
	"workouts upload": {
		args:   []string{"w.json"},
		files:  map[string]string{"w.json": `{"workoutName":"Intervals"}`},
		routes: map[string]string{"POST /workout-service/workout": `{"workoutId":8}`},
		json:   "{\n  \"workoutId\": 8\n}\n",
		human:  "Uploaded workout from $DIR/w.json\n",
	},
	"workouts update": {
		args:   []string{"7", "w.json"},
		files:  map[string]string{"w.json": `{"workoutId":8,"workoutName":"Intervals"}`},
		routes: map[string]string{"PUT /workout-service/workout/7": noContent},
		json:   okOut,
		notes:  "Workout JSON has workoutId 8, updating workout 7 instead\n",
		human:  "Updated workout 7 from $DIR/w.json\n",
	},
	"workouts create": {
		args:     []string{"Tempo", "--type", "run", "--step", "warmup:10m"},
		routes:   map[string]string{"POST /workout-service/workout": `{"workoutId":9}`},
		json:     "{\n  \"workoutId\": 9\n}\n",
		human:    "Created workout \"Tempo\" (ID: 9)\n",
		fallback: "Created workout \"Tempo\"\n",
	},
	"workouts copy": {
		args:     []string{"9", "Tempo B"},
		routes:   map[string]string{"POST /workout-service/workout/copy/9": `{"workoutId":10,"workoutName":"Tempo B"}`},
		json:     "{\n  \"workoutId\": 10,\n  \"workoutName\": \"Tempo B\"\n}\n",
		human:    "Copied workout 9 to \"Tempo B\" (ID: 10)\n",
		fallback: "Copied workout 9\n",
	},
	"workouts send": {
		args: []string{"9"},
		routes: map[string]string{
			"GET /workout-service/workout/9":                 `{"workoutName":"Tempo"}`,
			"GET /device-service/deviceregistration/devices": `[{"deviceId":5,"displayName":"Edge","workoutCapable":true}]`,
			"POST /device-service/devicemessage/messages":    `[{"messageId":7}]`,
		},
		result:   "POST /device-service/devicemessage/messages",
		json:     "[\n  {\n    \"messageId\": 7\n  }\n]\n",
		human:    "Sent workout \"Tempo\" to Edge (message 7). It reaches the device at its next sync. To cancel it before then, run: gccli devices messages cancel 7\n",
		fallback: "Sent workout \"Tempo\" to Edge. It reaches the device at its next sync. To cancel it before then, find the message in: gccli devices messages\n",
	},
	"workouts benchmarks set": {
		args: []string{"BARBELL_ROW", "--weight", "50", "--reps", "5"},
		routes: map[string]string{
			"GET /workout-service/benchmarks":  `[{"benchmarkCategoryKey":"ROW","benchmarkKey":"BARBELL_ROW"}]`,
			"POST /workout-service/benchmarks": `7`,
		},
		result:   "POST /workout-service/benchmarks",
		json:     "7\n",
		human:    "Set benchmark BARBELL_ROW to 50 kg x 5 (1RM 56.3 kg)\n",
		fallback: "Set benchmark BARBELL_ROW to 50 kg x 5 (1RM 56.3 kg)\n",
	},
	"workouts benchmarks clear": {
		args: []string{"BARBELL_ROW"},
		routes: map[string]string{
			"GET /workout-service/benchmarks":      `[{"id":7,"benchmarkCategoryKey":"ROW","benchmarkKey":"BARBELL_ROW"}]`,
			"DELETE /workout-service/benchmarks/7": noContent,
		},
		result:   "DELETE /workout-service/benchmarks/7",
		prompt:   "Clear benchmark BARBELL_ROW?",
		json:     okOut,
		human:    "Cleared benchmark BARBELL_ROW\n",
		fallback: "Cleared benchmark BARBELL_ROW\n",
	},
	"workouts schedule add": {
		args:   []string{"9", "2024-01-15"},
		routes: map[string]string{"POST /workout-service/schedule/9": `{"workoutScheduleId":10}`},
		json:   "{\n  \"workoutScheduleId\": 10\n}\n",
		human:  "Scheduled workout 9 on 2024-01-15\n",
	},
	"courses import": {
		args:  []string{"route.gpx"},
		files: map[string]string{"route.gpx": "<gpx/>"},
		routes: map[string]string{
			"POST /course-service/course/import":                          `{"courseName":"Loop","geoPoints":[{"latitude":1,"longitude":2}]}`,
			"POST /course-service/course/elevation?smoothingEnabled=true": `[[1,2,100]]`,
			"POST /course-service/course":                                 `{"courseId":42,"courseName":"Loop","distanceMeter":10000,"elevationGainMeter":100,"elevationLossMeter":90}`,
		},
		result:   "POST /course-service/course",
		json:     "{\n  \"courseId\": 42,\n  \"courseName\": \"Loop\",\n  \"distanceMeter\": 10000,\n  \"elevationGainMeter\": 100,\n  \"elevationLossMeter\": 90\n}\n",
		notes:    "Importing route.gpx...\n",
		fallback: "Imported course from route.gpx\n",
		table:    "ID  NAME  DISTANCE  ELEV GAIN  ELEV LOSS\n42  Loop  10.00 km  100 m      90 m\n",
		plain:    "42\tLoop\t10.00 km\t100 m\t90 m\n",
	},
	"courses send": {
		args: []string{"42", "99"},
		routes: map[string]string{
			"GET /course-service/course/42":               `{"courseName":"Loop"}`,
			"POST /device-service/devicemessage/messages": `[{"deviceName":"Edge","messageId":7}]`,
		},
		result:   "POST /device-service/devicemessage/messages",
		json:     "[\n  {\n    \"deviceName\": \"Edge\",\n    \"messageId\": 7\n  }\n]\n",
		human:    "Sent course \"Loop\" to device Edge (message 7)\n",
		fallback: "Sent course \"Loop\" to device 99\n",
	},
	"devices messages cancel": {
		args:   []string{"777"},
		routes: map[string]string{"DELETE /device-service/devicemessage/message/777": "true"},
		json:   "true\n",
		human:  "Cancelled message 777\n",
	},
	"devices set-primary-tracker": {
		args:   []string{"12345"},
		routes: map[string]string{"PUT /device-service/deviceservice/device-info/12345/primary-activity-tracker": noContent},
		json:   okOut,
		human:  "Set device 12345 as the primary activity tracker\n",
	},
	"events add": {
		args:     []string{"--name", "Race", "--date", "2024-06-01", "--type", "running"},
		routes:   map[string]string{"POST /calendar-service/event": `{"id":11,"eventName":"Race","date":"2024-06-01"}`},
		json:     "{\n  \"id\": 11,\n  \"eventName\": \"Race\",\n  \"date\": \"2024-06-01\"\n}\n",
		human:    "Added event \"Race\" on 2024-06-01 (ID: 11)\n",
		fallback: "Added event\n",
	},
	"segments favorite": {
		args: []string{"a1b2c3d4-0000-4000-8000-000000000001"},
		routes: map[string]string{
			"GET /segment-service/segment/a1b2c3d4-0000-4000-8000-000000000001": `{"segmentPk":900001,"uuid":"a1b2c3d4-0000-4000-8000-000000000001"}`,
			"PUT /segment-service/segment/user/900001":                          `{"segmentPk":900001,"favorite":true}`,
		},
		result:   "PUT /segment-service/segment/user/900001",
		json:     "{\n  \"segmentPk\": 900001,\n  \"favorite\": true\n}\n",
		human:    "Added segment a1b2c3d4-0000-4000-8000-000000000001 to favorites\n",
		fallback: "Added segment a1b2c3d4-0000-4000-8000-000000000001 to favorites\n",
	},
	"segments unfavorite": {
		args: []string{"a1b2c3d4-0000-4000-8000-000000000001"},
		routes: map[string]string{
			"GET /segment-service/segment/a1b2c3d4-0000-4000-8000-000000000001": `{"segmentPk":900001,"uuid":"a1b2c3d4-0000-4000-8000-000000000001"}`,
			"DELETE /segment-service/segment/user/900001":                       `{"segmentPk":900001,"favorite":false}`,
		},
		result:   "DELETE /segment-service/segment/user/900001",
		json:     "{\n  \"segmentPk\": 900001,\n  \"favorite\": false\n}\n",
		human:    "Removed segment a1b2c3d4-0000-4000-8000-000000000001 from favorites\n",
		fallback: "Removed segment a1b2c3d4-0000-4000-8000-000000000001 from favorites\n",
	},
	"segments create": {
		args: []string{"--from-activity", "5000001", "--start-km", "0", "--end-km", "0.1", "--name", "Park Hill", "--activity-type", "cycling"},
		routes: map[string]string{
			"GET /segment-service/segment/activity/5000001": `{"polyline":[{"distance":0,"elevation":10},{"distance":100,"elevation":12}]}`,
			"POST /segment-service/segment/":                `{"uuid":"a1b2c3d4-0000-4000-8000-000000000009"}`,
		},
		result:   "POST /segment-service/segment/",
		json:     "{\n  \"uuid\": \"a1b2c3d4-0000-4000-8000-000000000009\"\n}\n",
		human:    "Created segment \"Park Hill\" (UUID: a1b2c3d4-0000-4000-8000-000000000009)\n",
		fallback: "Created segment \"Park Hill\"\n",
	},
	"training task move": {
		args:     []string{"456", "789", "2026-10-28"},
		routes:   map[string]string{"POST /trainingplan-service/trainingplan/moveWorkoutTask/456": `{"taskWorkout":{"workoutScheduleId":790}}`},
		json:     "{\n  \"taskWorkout\": {\n    \"workoutScheduleId\": 790\n  }\n}\n",
		human:    "Moved task 789 to 2026-10-28 (new task ID: 790)\n",
		fallback: "Moved task 789 to 2026-10-28\n",
	},
	"calendar notes add": {
		args:     []string{"2026-10-20", "Long run", "--content", "Easy pace"},
		routes:   map[string]string{"POST /calendar-service/note": `{"id":118934,"noteName":"Long run","date":"2026-10-20"}`},
		json:     "{\n  \"id\": 118934,\n  \"noteName\": \"Long run\",\n  \"date\": \"2026-10-20\"\n}\n",
		human:    "Added note \"Long run\" on 2026-10-20 (ID: 118934)\n",
		fallback: "Added note \"Long run\" on 2026-10-20\n",
	},
	"calendar export enable": {
		routes: map[string]string{
			"GET /calendar-service/export":  `{"baseUrl":"http://localhost/proxy/calendar-service/export","id":"NOT_PUBLISHED"}`,
			"POST /calendar-service/export": `{"baseUrl":null,"id":"fedcba9876543210fedcba9876543210"}`,
		},
		result:   "POST /calendar-service/export",
		json:     "{\n  \"baseUrl\": null,\n  \"id\": \"fedcba9876543210fedcba9876543210\"\n}\n",
		fallback: "Turned on the calendar feed. Run gccli calendar export show to see its URL.\n",
		table:    "URL\nhttps://connect.garmin.com/modern/calendar/export/fedcba9876543210fedcba9876543210\n",
		plain:    "https://connect.garmin.com/modern/calendar/export/fedcba9876543210fedcba9876543210\n",
	},
	"calendar export disable": {
		routes: map[string]string{
			"GET /calendar-service/export":                                     `{"baseUrl":"http://localhost/proxy/calendar-service/export","id":"0123456789abcdef0123456789abcdef"}`,
			"DELETE /calendar-service/export/0123456789abcdef0123456789abcdef": noContent,
		},
		result:   "DELETE /calendar-service/export/0123456789abcdef0123456789abcdef",
		prompt:   "Turn off the calendar feed? Calendar apps that subscribe to it then get an empty answer.",
		json:     okOut,
		human:    "Turned off the calendar feed\n",
		fallback: "Turned off the calendar feed\n",
	},
	"calendar preferences set": {
		args: []string{"showWeeklyTotal", "false"},
		routes: map[string]string{
			"GET /calendar-service/preferences": `{"showWeeklyTotal":true,"view":1}`,
			"PUT /calendar-service/preferences": noContent,
		},
		result:   "PUT /calendar-service/preferences",
		json:     okOut,
		human:    "Set calendar preference showWeeklyTotal to false\n",
		fallback: "Set calendar preference showWeeklyTotal to false\n",
	},
	"calendar notes update": {
		args: []string{"118934", "--name", "Tempo run"},
		routes: map[string]string{
			"GET /calendar-service/note/118934": `{"id":118934,"noteName":"Long run","date":"2026-10-20"}`,
			"PUT /calendar-service/note/118934": noContent,
		},
		result:   "PUT /calendar-service/note/118934",
		json:     okOut,
		human:    "Updated note 118934\n",
		fallback: "Updated note 118934\n",
	},
	"challenges adhoc create": {
		args: []string{"Step up", "--type", "steps", "--start", "2026-10-19", "--invite", "ann"},
		routes: map[string]string{
			"GET /follower-service/follow/mutuals":          `[{"displayName":"ann","userId":2001}]`,
			"GET /userprofile-service/userprofile/settings": `{"displayName":"tester","profileId":3141592}`,
			"GET /userprofile-service/socialProfile/tester": `{"displayName":"tester","profileId":3141592}`,
			"POST /adhocchallenge-service/adHocChallenge":   `{"uuid":"FEDCBA9876543210FEDCBA9876543210"}`,
		},
		result:   "POST /adhocchallenge-service/adHocChallenge",
		json:     "{\n  \"uuid\": \"FEDCBA9876543210FEDCBA9876543210\"\n}\n",
		human:    "Created ad-hoc challenge \"Step up\" (UUID: FEDCBA9876543210FEDCBA9876543210)\n",
		fallback: "Created ad-hoc challenge \"Step up\"\n",
	},
	"challenges adhoc invite": {
		args: []string{"FEDCBA9876543210FEDCBA9876543210", "ann"},
		routes: map[string]string{
			"GET /follower-service/follow/mutuals":               `[{"displayName":"ann","userId":2001}]`,
			"GET /userprofile-service/userprofile/settings":      `{"displayName":"tester","profileId":3141592}`,
			"GET /userprofile-service/socialProfile/tester":      `{"displayName":"tester","profileId":3141592}`,
			"POST /adhocchallenge-service/adHocChallenge/invite": noContent,
		},
		result:   "POST /adhocchallenge-service/adHocChallenge/invite",
		json:     okOut,
		human:    "Invited 1 person to ad-hoc challenge FEDCBA9876543210FEDCBA9876543210\n",
		fallback: "Invited 1 person to ad-hoc challenge FEDCBA9876543210FEDCBA9876543210\n",
	},
	"gear add": {
		args:     []string{"Pegasus", "--brand", "Nike", "--model", "Pegasus", "--first-use", "2024-06-01"},
		routes:   map[string]string{"POST /gear-service/gear/v2": `{"uuid":"0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e","name":"Pegasus"}`},
		json:     "{\n  \"uuid\": \"0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e\",\n  \"name\": \"Pegasus\"\n}\n",
		human:    "Added gear \"Pegasus\" (UUID: 0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e)\n",
		fallback: "Added gear \"Pegasus\"\n",
	},
	"gear update": {
		args: []string{"0b5c3d2e1f4a4b6c8d7e9f0a1b2c3d4e", "--name", "Old"},
		routes: map[string]string{
			"GET /gear-service/gear/v2/0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e": `{"uuid":"0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e","name":"Pegasus"}`,
			"PUT /gear-service/gear/v2/0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e": noContent,
		},
		result:   "PUT /gear-service/gear/v2/0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e",
		json:     okOut,
		human:    "Updated gear 0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e\n",
		fallback: "Updated gear 0b5c3d2e-1f4a-4b6c-8d7e-9f0a1b2c3d4e\n",
	},
	"gear collections create": {
		args:     []string{"Road shoes", "--first-use", "2024-06-01"},
		routes:   map[string]string{"POST /gear-service/collection": `{"uuid":"7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b","name":"Road shoes"}`},
		json:     "{\n  \"uuid\": \"7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b\",\n  \"name\": \"Road shoes\"\n}\n",
		human:    "Created gear collection \"Road shoes\" (UUID: 7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b)\n",
		fallback: "Created gear collection \"Road shoes\"\n",
	},
	"gear collections rename": {
		args: []string{"7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b", "Trail shoes"},
		routes: map[string]string{
			"GET /gear-service/collection/7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b": `{"uuid":"7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b","name":"Road shoes"}`,
			"PUT /gear-service/collection/7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b": noContent,
		},
		result:   "PUT /gear-service/collection/7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b",
		json:     okOut,
		human:    "Renamed gear collection 7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b to \"Trail shoes\"\n",
		fallback: "Renamed gear collection 7a1e9c40-2b3d-4e5f-8a6b-9c0d1e2f3a4b to \"Trail shoes\"\n",
	},
	// The goals rows give no --date, so their routes pin the default date,
	// today on the harness clock.
	"goals set": {
		args:     []string{"floors-climbed", "10"},
		routes:   map[string]string{"PUT /goal-service/goal/user/effective/floors-climbed/2026-10-15": `{"goalType":"FLOORS_CLIMBED","goalValue":10}`},
		json:     "{\n  \"goalType\": \"FLOORS_CLIMBED\",\n  \"goalValue\": 10\n}\n",
		human:    "Set floors-climbed goal to 10 from 2026-10-15\n",
		fallback: "Set floors-climbed goal to 10 from 2026-10-15\n",
	},
	"goals add": {
		args:     []string{"Run 50 km", "--period", "seven-days", "--distance", "50km"},
		routes:   map[string]string{"POST /goal-service/goal/goal": `{"id":4711,"name":"Run 50 km"}`},
		json:     "{\n  \"id\": 4711,\n  \"name\": \"Run 50 km\"\n}\n",
		human:    "Added goal \"Run 50 km\" (ID: 4711)\n",
		fallback: "Added goal \"Run 50 km\"\n",
	},
	"goals update": {
		args: []string{"4711", "--name", "Run 60 km"},
		routes: map[string]string{
			"GET /goal-service/goal/goal/4711": `{"id":4711,"name":"Run 50 km"}`,
			"PUT /goal-service/goal/goal/4711": noContent,
		},
		result:   "PUT /goal-service/goal/goal/4711",
		json:     okOut,
		human:    "Updated goal 4711\n",
		fallback: "Updated goal 4711\n",
	},
	"goals weight set": {
		args:     []string{"72.5"},
		routes:   map[string]string{"PUT /goal-service/goal/user/effective/weight-grams/2026-10-15": `{"goalType":"WEIGHT_GRAMS","goalValue":72500.0}`},
		json:     "{\n  \"goalType\": \"WEIGHT_GRAMS\",\n  \"goalValue\": 72500.0\n}\n",
		human:    "Set weight goal to 72.5 kg from 2026-10-15\n",
		fallback: "Set weight goal to 72.5 kg from 2026-10-15\n",
	},
	"goals weight delete": {
		routes: map[string]string{
			"GET /goal-service/goal/user/WEIGHT_GRAMS/DAILY": `{"userGoalPk":42,"userGoalType":"WEIGHT_GRAMS","startDate":"2026-10-01","goalValue":72500.0}`,
			"DELETE /goal-service/goal/user/42":              noContent,
		},
		result:   "DELETE /goal-service/goal/user/42",
		prompt:   "Delete weight goal of 72.5 kg from 2026-10-01?",
		json:     okOut,
		human:    "Deleted weight goal from 2026-10-01\n",
		fallback: "Deleted weight goal from 2026-10-01\n",
	},
	"goals steps set": {
		args: []string{"9000"},
		routes: map[string]string{
			"GET /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-15": `{"userGoal":0,"userGoalCategory":"my-auto"}`,
			"PUT /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-15": `{"userGoal":9000,"userGoalCategory":"manual"}`,
		},
		result:   "PUT /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-15",
		json:     "{\n  \"userGoal\": 9000,\n  \"userGoalCategory\": \"manual\"\n}\n",
		human:    "Set steps goal to 9000 from 2026-10-15\n",
		fallback: "Set steps goal to 9000 from 2026-10-15\n",
	},
	"goals steps auto": {
		routes: map[string]string{
			"GET /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-15": `{"userGoal":9000,"userGoalCategory":"manual"}`,
			"PUT /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-15": `{"userGoal":0,"userGoalCategory":"my-auto"}`,
		},
		result:   "PUT /wellness-service/wellness/wellness-goals/consolidated/steps/2026-10-15",
		json:     "{\n  \"userGoal\": 0,\n  \"userGoalCategory\": \"my-auto\"\n}\n",
		human:    "Set automatic steps goal from 2026-10-15\n",
		fallback: "Set automatic steps goal from 2026-10-15\n",
	},
	"wellness jet-lag add": {
		args:     []string{"--from", "Berlin", "--to", "Tokyo", "--depart", "2026-11-02T08:00+01:00", "--arrive", "2026-11-02T23:30+09:00"},
		routes:   map[string]string{"POST /jetlag-service/jetlag/trip": `{"tripId":4711}`},
		json:     "{\n  \"tripId\": 4711\n}\n",
		human:    "Added jet lag trip from Berlin to Tokyo (ID: 4711)\n",
		fallback: "Added jet lag trip from Berlin to Tokyo\n",
	},
	"social follow": {
		args:     []string{"anna"},
		routes:   map[string]string{"POST /follower-service/follow/followings/anna": `{"id":7,"requestStatus":"CREATED"}`},
		json:     "{\n  \"id\": 7,\n  \"requestStatus\": \"CREATED\"\n}\n",
		human:    "Sent a follow request to anna\n",
		fallback: "Followed anna\n",
	},
	"social unfollow": {
		args: []string{"anna"},
		routes: map[string]string{
			"GET /follower-service/follow/anna/profileStatus": `{"userFollowProfile":{"followStatus":{"relationshipStatus":"FOLLOWING"}}}`,
			"DELETE /follower-service/follow/followings/anna": noContent,
		},
		result:   "DELETE /follower-service/follow/followings/anna",
		prompt:   "Unfollow anna?",
		json:     okOut,
		human:    "Unfollowed anna\n",
		fallback: "Unfollowed anna\n",
	},
	"social remove-follower": {
		args: []string{"anna"},
		routes: map[string]string{
			"GET /follower-service/follow/anna/profileStatus": `{"userFollowProfile":{"followStatus":{"relationshipStatus":"FOLLOWED"}}}`,
			"DELETE /follower-service/follow/followers/anna":  noContent,
		},
		result:   "DELETE /follower-service/follow/followers/anna",
		prompt:   "Remove anna from your followers?",
		json:     okOut,
		human:    "Removed follower anna\n",
		fallback: "Removed follower anna\n",
	},
	"social accept": {
		args: []string{"anna"},
		routes: map[string]string{
			"GET /web-gateway/inbox/v2/requests?req=8":       `{"requests":[{"data":{"displayName":"anna","followRequestId":7}}],"nextCursor":null}`,
			"PUT /follower-service/follow/requests/7/accept": noContent,
		},
		result:   "PUT /follower-service/follow/requests/7/accept",
		json:     okOut,
		human:    "Accepted the follow request of anna\n",
		fallback: "Accepted the follow request of anna\n",
	},
	"social decline": {
		args: []string{"anna"},
		routes: map[string]string{
			"GET /web-gateway/inbox/v2/requests?req=8":        `{"requests":[{"data":{"displayName":"anna","followRequestId":7}}],"nextCursor":null}`,
			"PUT /follower-service/follow/requests/7/decline": noContent,
		},
		result:   "PUT /follower-service/follow/requests/7/decline",
		json:     okOut,
		human:    "Declined the follow request of anna\n",
		fallback: "Declined the follow request of anna\n",
	},
	"social unmute": {
		args: []string{"anna"},
		routes: map[string]string{
			"GET /follower-service/muteuser":         `{"mutedUserList":[{"displayName":"anna"}],"mutedUserCount":1}`,
			"DELETE /follower-service/muteuser/anna": noContent,
		},
		result:   "DELETE /follower-service/muteuser/anna",
		json:     okOut,
		human:    "Unmuted anna\n",
		fallback: "Unmuted anna\n",
	},
	"wellness jet-lag update": {
		args: []string{"4711", "--to", "Osaka"},
		routes: map[string]string{
			"GET /jetlag-service/jetlag/trip/all": `[{"tripId":4711,"startDate":"2026-11-02","endDate":"2026-11-02","flights":[{"flightId":801,"originLocation":"Berlin","originTimestamp":"2026-11-02T08:00:00.0","originTimezone":"+01:00","destinationLocation":"Tokyo","destinationTimestamp":"2026-11-02T23:30:00.0","destinationTimezone":"+09:00"}]}]`,
			"PUT /jetlag-service/jetlag/trip":     noContent,
		},
		result:   "PUT /jetlag-service/jetlag/trip",
		json:     okOut,
		human:    "Updated jet lag trip 4711\n",
		fallback: "Updated jet lag trip 4711\n",
	},
	"wellness jet-lag delete": {
		args: []string{"4711"},
		routes: map[string]string{
			"GET /jetlag-service/jetlag/trip/all":     `[{"tripId":4711,"startDate":"2026-11-02","endDate":"2026-11-02","flights":[{"flightId":801,"originLocation":"Berlin","originTimestamp":"2026-11-02T08:00:00.0","originTimezone":"+01:00","destinationLocation":"Tokyo","destinationTimestamp":"2026-11-02T23:30:00.0","destinationTimezone":"+09:00"}]}]`,
			"DELETE /jetlag-service/jetlag/trip/4711": noContent,
		},
		result:   "DELETE /jetlag-service/jetlag/trip/4711",
		prompt:   "Delete jet lag trip 4711 (Berlin to Tokyo, 2026-11-02 to 2026-11-02)?",
		json:     okOut,
		human:    "Deleted jet lag trip 4711\n",
		fallback: "Deleted jet lag trip 4711\n",
	},
	"health sleep naps add": {
		args: []string{"09:00", "09:20"},
		routes: map[string]string{
			profileRoute:                    `{"displayName":"tester","timeZone":"gmt"}`,
			socialRoute:                     `{"profileId":3141592}`,
			"POST /sleep-service/sleep/nap": `{"calendarDate":"2026-10-15","napTimeSec":1200}`,
		},
		json:     "{\n  \"calendarDate\": \"2026-10-15\",\n  \"napTimeSec\": 1200\n}\n",
		human:    "Added nap on 2026-10-15 from 09:00 to 09:20\n",
		fallback: "Added nap on 2026-10-15 from 09:00 to 09:20\n",
	},
	"health sleep naps delete": {
		args: []string{"2026-10-15", "09:00"},
		routes: map[string]string{
			profileRoute: `{"displayName":"tester","timeZone":"gmt"}`,
			"GET /sleep-service/sleep/naps/2026-10-15":                 `[{"napStartTimestampGMT":"2026-10-15T09:00:00"}]`,
			"DELETE /sleep-service/sleep/nap/2026-10-15/1792054800000": noContent,
		},
		result:   "DELETE /sleep-service/sleep/nap/2026-10-15/1792054800000",
		prompt:   "Delete nap on 2026-10-15 at 09:00?",
		json:     okOut,
		human:    "Deleted nap on 2026-10-15 at 09:00\n",
		fallback: "Deleted nap on 2026-10-15 at 09:00\n",
	},
	"notifications dismiss": {
		args:   []string{"777"},
		routes: map[string]string{"PUT /notification-service/notification/777/dismiss/true": noContent},
		json:   okOut,
		human:  "Dismissed notification 777\n",
	},
	"notifications undismiss": {
		args:   []string{"777"},
		routes: map[string]string{"PUT /notification-service/notification/777/dismiss/false": noContent},
		json:   okOut,
		human:  "Restored notification 777\n",
	},
	"notifications preferences set": {
		args: []string{"allowEmailWeeklySummary", "false"},
		routes: map[string]string{
			"GET /notification-service/preference/userPreference": `{"allowEmailWeeklySummary":true,"allowPushBadges":false}`,
			"PUT /notification-service/preference/userPreference": "true",
		},
		result:   "PUT /notification-service/preference/userPreference",
		json:     "true\n",
		human:    "Set notification preference allowEmailWeeklySummary to false\n",
		fallback: "Set notification preference allowEmailWeeklySummary to false\n",
	},
	"profile privacy-zones add": {
		args:     []string{"--lat", "52.52", "--lon=-30.5", "--radius", "800"},
		routes:   map[string]string{"POST /privacyzone-service/privacyZones": `{"centerLatitude":52.52,"centerLongitude":-30.5,"privacyZoneId":4321,"radius":800}`},
		json:     "{\n  \"centerLatitude\": 52.52,\n  \"centerLongitude\": -30.5,\n  \"privacyZoneId\": 4321,\n  \"radius\": 800\n}\n",
		human:    "Added privacy zone of 800 m around 52.52, -30.5 (ID: 4321)\n",
		fallback: "Added privacy zone of 800 m around 52.52, -30.5\n",
	},
	"wellness menstrual-log set": {
		args: []string{"2026-10-01", "--flow", "light"},
		routes: map[string]string{
			"GET /periodichealth-service/menstrualcycle/dailylog/2026-10-01":  noContent,
			"POST /periodichealth-service/menstrualcycle/dailylog/2026-10-01": `{"calendarDate":"2026-10-01","flow":"LIGHT"}`,
		},
		result:   "POST /periodichealth-service/menstrualcycle/dailylog/2026-10-01",
		json:     "{\n  \"calendarDate\": \"2026-10-01\",\n  \"flow\": \"LIGHT\"\n}\n",
		human:    "Saved menstrual log for 2026-10-01\n",
		fallback: "Saved menstrual log for 2026-10-01\n",
	},
	"wellness menstrual-log delete": {
		args: []string{"2026-10-01"},
		routes: map[string]string{
			"GET /periodichealth-service/menstrualcycle/dailylog/2026-10-01":    `{"calendarDate":"2026-10-01","flow":"LIGHT"}`,
			"DELETE /periodichealth-service/menstrualcycle/dailylog/2026-10-01": noContent,
		},
		result:   "DELETE /periodichealth-service/menstrualcycle/dailylog/2026-10-01",
		prompt:   "Delete menstrual log for 2026-10-01?",
		json:     okOut,
		human:    "Deleted menstrual log for 2026-10-01\n",
		fallback: "Deleted menstrual log for 2026-10-01\n",
	},
	"nutrition add": {
		args: []string{"2026-10-15", "1001", "--query", "apple", "--meal", "snacks"},
		routes: map[string]string{
			"GET /nutrition-service/food/search?searchExpression=apple&start=0&limit=50": `{"results":[{"foodMetaData":{"foodId":"1001","foodName":"Apple","source":"FATSECRET","regionCode":"US","languageCode":"en"},"nutritionContents":[{"servingId":"2001","servingUnit":"medium"}]}]}`,
			"GET /nutrition-service/meals/2026-10-15":                                    `{"meals":[{"mealId":105,"mealName":"SNACKS","enabled":true}]}`,
			"PUT /nutrition-service/food/logs":                                           `{"mealDetails":[{"loggedFoods":[{"logId":"a1","logTimestamp":"2026-10-15T09:30:00.000Z"}]}]}`,
		},
		result:   "PUT /nutrition-service/food/logs",
		json:     "{\n  \"mealDetails\": [\n    {\n      \"loggedFoods\": [\n        {\n          \"logId\": \"a1\",\n          \"logTimestamp\": \"2026-10-15T09:30:00.000Z\"\n        }\n      ]\n    }\n  ]\n}\n",
		human:    "Added Apple (1 x medium) to snacks on 2026-10-15 (log ID: a1)\n",
		fallback: "Added Apple (1 x medium) to snacks on 2026-10-15\n",
	},
	"nutrition delete": {
		args: []string{"2026-10-15", "a1"},
		routes: map[string]string{
			"GET /nutrition-service/food/logs/2026-10-15":    `{"mealDetails":[{"loggedFoods":[{"logId":"a1"}]}]}`,
			"DELETE /nutrition-service/food/logs/2026-10-15": noContent,
		},
		result:   "DELETE /nutrition-service/food/logs/2026-10-15",
		prompt:   "Delete food log entry a1 on 2026-10-15?",
		json:     okOut,
		human:    "Deleted food log entry a1 on 2026-10-15\n",
		fallback: "Deleted food log entry a1 on 2026-10-15\n",
	},
	"nutrition favorites add": {
		args: []string{"1001", "--query", "apple"},
		routes: map[string]string{
			"GET /nutrition-service/food/search?searchExpression=apple&start=0&limit=50": `{"results":[{"foodMetaData":{"foodId":"1001","foodName":"Apple","source":"FATSECRET","regionCode":"US","languageCode":"en"},"nutritionContents":[{"servingId":"2001","servingUnit":"medium"}]}]}`,
			"PUT /nutrition-service/favorite/food":                                       `{"foodId":"1001"}`,
		},
		result:   "PUT /nutrition-service/favorite/food",
		json:     "{\n  \"foodId\": \"1001\"\n}\n",
		human:    "Added Apple (1 x medium) to favorite foods\n",
		fallback: "Added Apple (1 x medium) to favorite foods\n",
	},
	"nutrition favorites remove": {
		args: []string{"1001"},
		routes: map[string]string{
			"GET /nutrition-service/favorite?start=0&limit=50&includeContent=true": `{"consumables":[{"foodMetaData":{"foodId":"1001"}}],"hasMore":false}`,
			"DELETE /nutrition-service/favorite/food/1001":                         noContent,
		},
		result:   "DELETE /nutrition-service/favorite/food/1001",
		json:     okOut,
		human:    "Removed food 1001 from favorite foods\n",
		fallback: "Removed food 1001 from favorite foods\n",
	},
	"nutrition quick-add": {
		args: []string{"2026-10-15", "--meal", "snacks", "--calories", "120"},
		routes: map[string]string{
			"GET /nutrition-service/meals/2026-10-15":   `{"meals":[{"mealId":105,"mealName":"SNACKS","enabled":true}]}`,
			"PUT /nutrition-service/food/logs/quickAdd": `{"mealDetails":[{"loggedFoods":[{"logId":"b2","logTimestamp":"2026-10-15T09:30:00.000Z"}]}]}`,
		},
		result:   "PUT /nutrition-service/food/logs/quickAdd",
		json:     "{\n  \"mealDetails\": [\n    {\n      \"loggedFoods\": [\n        {\n          \"logId\": \"b2\",\n          \"logTimestamp\": \"2026-10-15T09:30:00.000Z\"\n        }\n      ]\n    }\n  ]\n}\n",
		human:    "Added 120 kcal to snacks on 2026-10-15 (log ID: b2)\n",
		fallback: "Added 120 kcal to snacks on 2026-10-15\n",
	},
	"hydration add": {
		args:   []string{"250", "--date", "2024-01-15"},
		routes: map[string]string{"PUT /usersummary-service/usersummary/hydration/log": `{"valueInML":250}`},
		json:   "{\n  \"valueInML\": 250\n}\n",
		human:  "Added 250 ml of water on 2024-01-15\n",
	},
}

type writeMode struct {
	name  string
	flags []string
	empty bool
}

var writeModes = []writeMode{
	{"json", []string{"--json"}, false},
	{"json-empty", []string{"--json"}, true},
	{"table", nil, false},
	{"table-empty", nil, true},
	{"plain", []string{"--plain"}, false},
}

func (tc writeCase) want(m writeMode) (stdout, stderr string) {
	switch m.name {
	case "json":
		return tc.json, tc.notes
	case "json-empty":
		return okOut, tc.notes
	case "table":
		return tc.table, tc.notes + tc.human
	case "table-empty":
		return "", tc.notes + tc.fallback
	default:
		return tc.plain, tc.notes + tc.human
	}
}

func tableWriteCases(t *testing.T) map[string]writeCase {
	t.Helper()
	cases := map[string]writeCase{}
	walkNodes(commands, "", func(path string, n Node) {
		l, ok := n.(Leaf)
		if !ok {
			return
		}
		w, ok := l.Output.(Write)
		if !ok {
			return
		}
		g, err := readGolden(goldenFile(path))
		if err != nil {
			t.Fatal(err)
		}
		args, in, err := g.input()
		if err != nil {
			t.Fatalf("%s: %v", goldenFile(path), err)
		}
		tc := writeCase{args: args[len(strings.Fields(path)):], files: in.files, routes: map[string]string{}}
		for route, reply := range in.routes {
			tc.routes[route] = reply.body
		}
		tc.result = tc.resultRoute(t)
		tc.json = g.sections["json stdout"]
		tc.human = g.sections["table stderr"]
		if w.Confirm != "" {
			tc.prompt, tc.human, _ = strings.Cut(tc.human, " [y/N]: ")
		}
		cases[path] = tc
	})
	return cases
}

func TestTableWriteCasesCoverWriteRows(t *testing.T) {
	var want []string
	walkNodes(commands, "", func(path string, n Node) {
		if l, ok := n.(Leaf); ok {
			if _, ok := l.Output.(Write); ok {
				want = append(want, path)
			}
		}
	})
	slices.Sort(want)
	got := slices.Sorted(maps.Keys(tableWriteCases(t)))
	if !slices.Equal(got, want) {
		t.Errorf("tableWriteCases covers %q\nwant the Leaf rows with a Write output %q", got, want)
	}
}

func allWriteCases(t *testing.T) map[string]writeCase {
	t.Helper()
	cases := maps.Clone(writeCases)
	for path, tc := range tableWriteCases(t) {
		if _, dup := cases[path]; dup {
			t.Errorf("writeCases[%q] names a Leaf row: remove the row", path)
		}
		cases[path] = tc
	}
	return cases
}

func TestWriteCommandOutput(t *testing.T) {
	cases := allWriteCases(t)
	for _, path := range slices.Sorted(maps.Keys(cases)) {
		tc := cases[path]
		for _, m := range writeModes {
			if m.name == "table-empty" && tc.fallback == "" {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				tc := tc
				if m.empty {
					tc.routes = maps.Clone(tc.routes)
					tc.routes[tc.resultRoute(t)] = noContent
				}
				h := newHarness(t, tc.input())
				args := append(strings.Fields(path), tc.args...)
				if tc.prompt != "" {
					args = append(args, "--force")
				}
				stdout, stderr, code := h.run(m.flags, args)
				wantOut, wantErr := tc.want(m)
				wantOut, wantErr = strings.ReplaceAll(wantOut, "$DIR", h.dir), strings.ReplaceAll(wantErr, "$DIR", h.dir)
				if code != 0 || stdout != wantOut || stderr != wantErr {
					t.Errorf("exit %d, want 0\nstdout %q\nwant   %q\nstderr %q\nwant   %q", code, stdout, wantOut, stderr, wantErr)
				}
				if len(h.writes()) == 0 {
					t.Error("no write request reached the mock server")
				}
			})
		}
	}
}

func TestWriteCommandDeclined(t *testing.T) {
	cases := allWriteCases(t)
	for _, path := range slices.Sorted(maps.Keys(cases)) {
		tc := cases[path]
		if tc.prompt == "" {
			continue
		}
		for _, m := range writeModes {
			if m.empty {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				in := tc.input()
				in.stdin = "n\n"
				h := newHarness(t, in)

				stdout, stderr, code := h.run(m.flags, append(strings.Fields(path), tc.args...))
				want := tc.prompt + " [y/N]: Cancelled\n"
				if code != 0 || stdout != "" || stderr != want {
					t.Errorf("exit %d, want 0\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
				}
				if w := h.writes(); len(w) != 0 {
					t.Errorf("declined prompt sent writes: %v", w)
				}
			})
		}
	}
}

func TestWriteCommandNotInteractive(t *testing.T) {
	const msg = "stdin is not a terminal: use --force to confirm"
	cases := allWriteCases(t)
	for _, path := range slices.Sorted(maps.Keys(cases)) {
		tc := cases[path]
		if tc.prompt == "" {
			continue
		}
		for _, m := range writeModes {
			if m.empty {
				continue
			}
			t.Run(path+"/"+m.name, func(t *testing.T) {
				h := newHarness(t, tc.input())
				useNonTerminalStdin(t)

				stdout, stderr, code := h.run(m.flags, append(strings.Fields(path), tc.args...))
				want := msg + "\n"
				if m.name == "json" {
					want = `{"error":{"code":"error","message":"` + msg + `"}}` + "\n"
				}
				if code != 1 || stdout != "" || stderr != want {
					t.Errorf("exit %d, want 1\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
				}
				if w := h.writes(); len(w) != 0 {
					t.Errorf("non-interactive run sent writes: %v", w)
				}
			})
		}
	}
}

func TestWriteCommandNotJSON(t *testing.T) {
	tc := writeCases["activity rename"]
	tc.routes = map[string]string{tc.resultRoute(t): "<html>saved</html>"}
	h := newHarness(t, tc.input())

	stdout, stderr, code := h.run([]string{"--json"}, append([]string{"activity", "rename"}, tc.args...))
	want := `{"error":{"code":"error","message":"garmin saved the change, but its response is not JSON: \"<html>saved</html>\""}}` + "\n"
	if code != 1 || stdout != "" || stderr != want {
		t.Errorf("exit %d, want 1\nstdout %q, want empty\nstderr %q\nwant   %q", code, stdout, stderr, want)
	}
}

func (tc writeCase) resultRoute(t *testing.T) string {
	t.Helper()
	if tc.result != "" {
		return tc.result
	}
	var data []string
	for route := range tc.routes {
		if !isLookup(route) {
			data = append(data, route)
		}
	}
	if len(data) != 1 {
		t.Fatalf("the case has %d data routes %q: set result", len(data), data)
	}
	return data[0]
}

func TestResultRouteSkipsLookups(t *testing.T) {
	tc := writeCase{routes: map[string]string{profileRoute: profileBody, socialRoute: profileBody, "DELETE /x/1": noContent}}
	for range 20 {
		if got := tc.resultRoute(t); got != "DELETE /x/1" {
			t.Fatalf("resultRoute = %q, want the one data route", got)
		}
	}
}

type mockReply struct {
	status int
	body   string
}

type harnessInput struct {
	files  map[string]string
	routes map[string]mockReply
	stdin  string
}

func (tc writeCase) input() harnessInput {
	routes := map[string]mockReply{}
	for route, body := range tc.routes {
		reply := mockReply{body: body}
		if body == noContent {
			reply.status = http.StatusNoContent
		}
		routes[route] = reply
	}
	return harnessInput{files: tc.files, routes: routes}
}

var harnessNow = time.Date(2026, 10, 15, 9, 30, 0, 0, time.UTC)

const harnessAccount = "test@example.com"

func harnessTokens(email string) *garminauth.Tokens {
	return &garminauth.Tokens{
		OAuth2AccessToken: "test-access-token",
		OAuth2ExpiresAt:   time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		Domain:            "garmin.com",
		Email:             email,
	}
}

type harness struct {
	dir  string
	home string
	mu   sync.Mutex
	log  []loggedRequest
}

type loggedRequest struct {
	method string
	route  string
	entry  string
}

func newHarness(t *testing.T, in harnessInput) *harness {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := t.TempDir()
	t.Chdir(dir)
	for name, content := range in.files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	h := &harness{dir: dir, home: home}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := r.Method + " " + r.URL.RequestURI()
		entry, err := requestlog.Format(r)
		if err != nil {
			t.Errorf("%s: %v", route, err)
		}
		h.mu.Lock()
		h.log = append(h.log, loggedRequest{r.Method, route, entry})
		h.mu.Unlock()
		reply, ok := in.routes[route]
		switch {
		case !ok:
			t.Errorf("unexpected request %s", route)
			w.WriteHeader(http.StatusNotFound)
		default:
			if reply.body != "" {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(cmp.Or(reply.status, http.StatusOK))
			_, _ = w.Write([]byte(reply.body))
		}
	}))
	t.Cleanup(server.Close)

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, harnessAccount, harnessTokens(harnessAccount))

	stub(t, &nowFn, func() time.Time { return harnessNow })
	stub(t, &confirmReader, io.Reader(strings.NewReader(in.stdin)))
	stub(t, &readStdinFn, func() ([]byte, error) { return []byte(in.stdin), nil })
	stub(t, &readPasswordFn, func(int) ([]byte, error) { return []byte("secret"), nil })
	stub(t, &loginBrowserFn, func(_ context.Context, email string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return harnessTokens(email), nil
	})
	stub(t, &loginHeadlessFn, func(_ context.Context, email, _ string, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return harnessTokens(email), nil
	})
	stub(t, &refreshFn, func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, errors.New("the test harness does not refresh tokens")
	})
	// Mirrors garminapi.FetchExerciseCatalog, whose URL is fixed, so that the
	// golden files show the error text of the binary.
	stub(t, &fetchExerciseCatalogFn, func(ctx context.Context) (json.RawMessage, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/web-data/exercises/Exercises.json", nil)
		if err != nil {
			return nil, err
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch exercise catalog: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return nil, fmt.Errorf("fetch exercise catalog: HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	})
	return h
}

func stub[T any](t *testing.T, v *T, fake T) {
	orig := *v
	*v = fake
	t.Cleanup(func() { *v = orig })
}

func (h *harness) run(mode, args []string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	global := append([]string{"--account", harnessAccount, "--color", "never", "--units", "metric"}, mode...)
	code = execute(append(global, args...), "test", "none", "never", &out, &errOut)
	return out.String(), errOut.String(), code
}

func (h *harness) writes() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var writes []string
	for _, r := range h.log {
		if r.method != http.MethodGet {
			writes = append(writes, r.route)
		}
	}
	return writes
}

func (h *harness) requests() []loggedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.log)
}

func TestWriteCensus(t *testing.T) {
	fset := token.NewFileSet()
	cli := parseFuncs(t, fset, ".")
	methods := clientMethods(parseFuncs(t, fset, "../garminapi"))
	apiWrites := apiWriteMethods(methods, tableWrites(garminapi.Describe, cli, slices.Collect(maps.Values(methods))))
	for name := range apiWrites {
		fn, ok := methods[name]
		if !ok {
			continue
		}
		if res := fn.Type.Results; ast.IsExported(name) && (res == nil || len(res.List) != 2 ||
			types.ExprString(res.List[0].Type) != "json.RawMessage" || types.ExprString(res.List[1].Type) != "error") {
			t.Errorf("%s: Client.%s sends a write: return (json.RawMessage, error) so the command can print the response", fset.Position(fn.Pos()), name)
		}
	}

	paths := map[string][]string{}
	commandPaths(newTestParser(t).Model.Node, "", paths)
	banned := []string{"IsJSON", "ModeFromContext", "WriteJSON", "WriteRows", "writeObject", "Stdout", "Successf"}
	found := map[string]bool{}
	for fn, method := range runWrites(cli, apiWrites) {
		pos := fset.Position(fn.Pos())
		recv := recvName(fn)
		if len(paths[recv]) == 0 {
			t.Errorf("%s: %s.Run sends a write but is not a command in the CLI tree", pos, recv)
		}
		ids, called := idents(fn.Body), calls(fn.Body)
		for _, path := range paths[recv] {
			found[path] = true
			if _, ok := writeCases[path]; !ok {
				t.Errorf("%s: %q sends a write to Garmin through Client.%s: add a row to writeCases in write_test.go", pos, path, method)
			}
			if !called["reportWrite"] && !called["reportWriteTable"] {
				t.Errorf("%s: %q sends a write: end Run with reportWrite or reportWriteTable (write.go)", pos, path)
			}
			for _, name := range banned {
				if ids[name] {
					t.Errorf("%s: %q names %s: print the write result only through reportWrite or reportWriteTable", pos, path, name)
				}
			}
		}
	}
	for path := range writeCases {
		if !found[path] {
			t.Errorf("writeCases[%q] in write_test.go names no command that sends a write: remove the row", path)
		}
	}
}

func TestWriteCensusFindsIndirectWrites(t *testing.T) {
	fset := token.NewFileSet()
	api := parseSource(t, fset, `package garminapi
func (c *Client) PatchNote(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, "PATCH", "/note", nil)
}
func (c *Client) SaveNote(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/note", nil)
}
func (c *Client) GetNote(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/note", nil)
}`)
	cli := parseSource(t, fset, `package cmd
func (c *LiteralVerbCmd) Run(g *Globals) error {
	_, err := client.PatchNote(g.Context)
	return err
}
func (c *HelperCmd) Run(g *Globals) error { return c.save(g) }
func (c *HelperCmd) save(g *Globals) error { return storeNote(g) }
func storeNote(g *Globals) error {
	_, err := client.SaveNote(g.Context)
	return err
}
func (c *ReadCmd) Run(g *Globals) error {
	_, err := client.GetNote(g.Context)
	return err
}`)

	got := map[string]string{}
	for fn, method := range runWrites(cli, apiWriteMethods(clientMethods(api), nil)) {
		got[recvName(fn)] = method
	}
	want := map[string]string{"LiteralVerbCmd": "PatchNote", "HelperCmd": "SaveNote"}
	if !maps.Equal(got, want) {
		t.Errorf("runWrites = %v, want %v", got, want)
	}
}

func TestWriteCensusFindsCallWrites(t *testing.T) {
	fset := token.NewFileSet()
	api := parseSource(t, fset, `package garminapi
func (c *Client) RenameNote(ctx context.Context, id string) (json.RawMessage, error) {
	body, err := json.Marshal(id)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "notes.update", map[string]string{"id": id}, body)
}`)
	cli := parseSource(t, fset, `package cmd
func (c *DeleteCmd) Run(g *Globals) error {
	_, err := client.Call(g.Context, "notes.delete", map[string]string{"id": c.ID}, nil)
	return err
}
func (c *HelperCmd) Run(g *Globals) error { return c.rename(g) }
func (c *HelperCmd) rename(g *Globals) error {
	_, err := client.RenameNote(g.Context, c.ID)
	return err
}
func (c *ReadCmd) Run(g *Globals) error {
	_, err := client.Call(g.Context, "notes.get", map[string]string{"id": c.ID}, nil)
	return err
}
func (c *SearchCmd) Run(g *Globals) error {
	_, err := client.Call(g.Context, "notes.search", nil, body)
	return err
}`)
	table := map[string]garminapi.Endpoint{
		"notes.delete": {Method: http.MethodDelete},
		"notes.update": {Method: http.MethodPut},
		"notes.get":    {Method: http.MethodGet},
		"notes.search": {Method: http.MethodPost, Body: true, Read: true},
	}
	lookup := func(name string) (garminapi.Endpoint, bool) {
		e, ok := table[name]
		return e, ok
	}

	got := map[string]string{}
	methods := clientMethods(api)
	for fn, method := range runWrites(cli, apiWriteMethods(methods, tableWrites(lookup, cli, api))) {
		got[recvName(fn)] = method
	}
	want := map[string]string{"DeleteCmd": `Call("notes.delete")`, "HelperCmd": "RenameNote"}
	if !maps.Equal(got, want) {
		t.Errorf("runWrites = %v, want %v", got, want)
	}
}

func clientMethods(fns []*ast.FuncDecl) map[string]*ast.FuncDecl {
	methods := map[string]*ast.FuncDecl{}
	for _, fn := range fns {
		if recvName(fn) == "Client" {
			methods[fn.Name.Name] = fn
		}
	}
	return methods
}

type clientMethod = string

func apiWriteMethods(methods map[string]*ast.FuncDecl, table map[string]clientMethod) map[string]clientMethod {
	writes := map[string]clientMethod{}
	maps.Copy(writes, table)
	for name, fn := range methods {
		if namesWriteVerb(fn.Body) {
			writes[name] = name
		}
	}
	return reachWrites(slices.Collect(maps.Values(methods)), writes)
}

func runWrites(fns []*ast.FuncDecl, apiWrites map[string]clientMethod) map[*ast.FuncDecl]clientMethod {
	writes := map[string]clientMethod{}
	for name := range apiWrites {
		writes[name] = name
	}
	var helpers, runs []*ast.FuncDecl
	for _, fn := range fns {
		if fn.Name.Name == "Run" && fn.Recv != nil {
			runs = append(runs, fn)
		} else {
			helpers = append(helpers, fn)
		}
	}
	writes = reachWrites(helpers, writes)
	out := map[*ast.FuncDecl]clientMethod{}
	for _, fn := range runs {
		if callee, ok := firstCallIn(fn.Body, writes); ok {
			out[fn] = writes[callee]
		}
	}
	return out
}

func reachWrites(fns []*ast.FuncDecl, writes map[string]clientMethod) map[string]clientMethod {
	for grew := true; grew; {
		grew = false
		for _, fn := range fns {
			if _, ok := writes[fn.Name.Name]; ok {
				continue
			}
			if callee, ok := firstCallIn(fn.Body, writes); ok {
				writes[fn.Name.Name], grew = writes[callee], true
			}
		}
	}
	return writes
}

func firstCallIn(body ast.Node, writes map[string]clientMethod) (string, bool) {
	for _, name := range slices.Sorted(maps.Keys(calls(body))) {
		if _, ok := writes[name]; ok {
			return name, true
		}
	}
	return "", false
}

var writeVerbs = map[string]bool{
	"MethodPost": true, "MethodPut": true, "MethodPatch": true, "MethodDelete": true,
	`"POST"`: true, `"PUT"`: true, `"PATCH"`: true, `"DELETE"`: true,
}

func namesWriteVerb(body ast.Node) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			found = found || writeVerbs[n.Name]
		case *ast.BasicLit:
			found = found || (n.Kind == token.STRING && writeVerbs[n.Value])
		}
		return !found
	})
	return found
}

func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return ""
	}
	return strings.TrimPrefix(types.ExprString(fn.Recv.List[0].Type), "*")
}

func parseFuncs(t *testing.T, fset *token.FileSet, dir string) []*ast.FuncDecl {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var fns []*ast.FuncDecl
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		fns = append(fns, funcDecls(f)...)
	}
	return fns
}

func parseSource(t *testing.T, fset *token.FileSet, src string) []*ast.FuncDecl {
	t.Helper()
	f, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	return funcDecls(f)
}

func funcDecls(f *ast.File) []*ast.FuncDecl {
	var fns []*ast.FuncDecl
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
			fns = append(fns, fn)
		}
	}
	return fns
}

func idents(n ast.Node) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			out[id.Name] = true
		}
		return true
	})
	return out
}

func calls(n ast.Node) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(n, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				out[fun.Name] = true
			case *ast.SelectorExpr:
				out[fun.Sel.Name] = true
				if name, ok := endpointName(call); ok {
					out[callKey(name)] = true
				}
			}
		}
		return true
	})
	return out
}

func callKey(endpoint string) string { return `Call("` + endpoint + `")` }

func endpointName(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Call" || len(call.Args) != 4 {
		return "", false
	}
	lit, ok := call.Args[1].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	name, err := strconv.Unquote(lit.Value)
	return name, err == nil
}

func tableWrites(lookup func(string) (garminapi.Endpoint, bool), fnSets ...[]*ast.FuncDecl) map[string]clientMethod {
	writes := map[string]clientMethod{}
	for _, fns := range fnSets {
		for _, fn := range fns {
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := endpointName(call)
				if !ok {
					return true
				}
				if e, known := lookup(name); known && e.Method != http.MethodGet && !e.Read {
					writes[callKey(name)] = callKey(name)
				}
				return true
			})
		}
	}
	return writes
}

func commandPaths(n *kong.Node, prefix string, out map[string][]string) {
	for _, child := range n.Children {
		path := prefix
		if child.Type == kong.CommandNode {
			path = strings.TrimSpace(prefix + " " + child.Name)
			name := reflect.Indirect(child.Target).Type().Name()
			out[name] = append(out[name], path)
		}
		commandPaths(child, path, out)
	}
}
