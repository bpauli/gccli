package conformance

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Record is one Garmin API response. Body is set only for a JSON response.
// For any other non-empty response, NonJSONBytes gives its size and the
// bytes are not kept.
type Record struct {
	Method       string          `json:"method"`
	Host         string          `json:"host"`
	URI          string          `json:"uri"`
	Status       int             `json:"status"`
	Body         json.RawMessage `json:"body,omitempty"`
	NonJSONBytes int             `json:"nonJSONBytes,omitempty"`
}

func EncodeRecord(r Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeRecord parses the file content of a record. It refuses unknown
// fields, so a file that holds more than a Record, such as headers, fails.
func DecodeRecord(data []byte) (Record, error) {
	var r Record
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return Record{}, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return Record{}, errors.New("data after the record")
	}
	return r, nil
}

var garminHosts = []string{
	"connectapi.garmin.com", "connect.garmin.com",
	"connectapi.garmin.cn", "connect.garmin.cn",
}

var scrubPaths = []string{
	"/activitylist-service/activities/count",
	"/activitylist-service/activities/search/activities",
	"/activity-service/activity",
	"/activity-service/activity/{activityId}",
	"/activity-service/activity/{activityId}/exerciseSets",
	"/activity-service/activity/{activityId}/hrTimeInZones",
	"/activity-service/activity/{activityId}/splits",
	"/activity-service/activity/{activityId}/weather",
	"/calendar-service/event",
	"/calendar-service/event/{eventId}",
	"/calendar-service/events",
	"/calendar-service/year/{year}/month/{month}/day/{day}/start/{start}",
	"/course-service/course",
	"/course-service/course/elevation",
	"/course-service/course/favorites",
	"/course-service/course/import",
	"/course-service/course/{courseId}",
	"/device-service/deviceregistration/devices",
	"/goal-service/goal/goals",
	"/personalrecord-service/personalrecord/prs/{displayName}",
	"/userprofile-service/userprofile/settings",
	"/userprofile-service/userprofile/user-settings",
	"/userstats-service/wellness/daily/{displayName}",
	"/usersummary-service/usersummary/daily/{displayName}",
	"/web-gateway/course/owner/",
	"/wellness-service/wellness/dailyHeartRate/{displayName}",
	"/wellness-service/wellness/dailySleepData/{displayName}",
	"/wellness-service/wellness/dailySummaryChart/{displayName}",
	"/workout-service/schedule/{id}",
	"/workout-service/workout",
	"/workout-service/workout/FIT/{workoutId}",
	"/workout-service/workout/{workoutId}",
	"/workout-service/workouts",
}

type kind uint8

const (
	kString kind = 1 << iota
	kNumber
	kBool
	kNull
	kObject
	kArray
)

func (k kind) String() string {
	switch k {
	case kString:
		return "string"
	case kNumber:
		return "number"
	case kBool:
		return "bool"
	case kNull:
		return "null"
	case kObject:
		return "object"
	}
	return "array"
}

type rule struct {
	kinds                kind
	key, jsonPath, value string
	to                   func(any) any
}

const (
	testEmail       = "t@example.com"
	testDisplayName = "tester"
	testName        = "Test User"
	testText        = "Test"
	testID          = "test-id"
	testSerial      = "TEST-SERIAL"
	testToken       = "test-token"
	testURL         = "https://example.com/test"
	testTimeZone    = "UTC"
	testGender      = "MALE"
)

const (
	enumName      = `(?i)(key|type|status|system|format|unit|level|order|label|category|point|manufacturer)$`
	idName        = `(Id|ID|Uuid|UUID|Guid|GUID|Pk|PK)$|^(id|uuid|guid|pk)$`
	freeTextName  = `^(activityName|courseName|workoutName|eventName|title|name|description|desc|notes?|comments?|location|locationName)$`
	enumWord      = `^[A-Za-z][A-Za-z0-9]*([_-][A-Za-z0-9]+)*$`
	dateOrTime    = `^(\d{4}-\d{2}-\d{2}([T ]\d{2}:\d{2}(:\d{2}(\.\d+)?)?)?|\d{2}:\d{2}(:\d{2})?)(Z|[+-]\d{2}:?\d{2})?$`
	timeZoneValue = `^[A-Z][a-z]+(/[A-Za-z_]+)+$`
)

var scrubRules = []rule{
	{kinds: kNumber, to: fixedNumber},
	{kinds: kBool | kNull, to: keep},
	{kinds: kString, value: `^$`, to: keep},
	{kinds: kString, key: `(?i)(token|secret|password|cookie|session|ticket|signature|credential|apikey|api_key)`, to: fixed(testToken)},
	{kinds: kString, value: `^[^@\s]+@[^@\s]+$`, to: fixed(testEmail)},
	{kinds: kString, key: `(?i)e-?mail`, to: fixed(testEmail)},
	{kinds: kString, value: `^[a-z][a-z0-9+.-]*://`, to: fixed(testURL)},
	{kinds: kString, key: `(?i)url$`, to: fixed(testURL)},
	{kinds: kString, key: `^(displayName|ownerDisplayName|userDisplayName)$`, to: fixed(testDisplayName)},
	{kinds: kString, key: `^(userName|fullName|firstName|lastName|ownerFullName|userFullName)$`, to: fixed(testName)},
	{kinds: kString, key: `(?i)serial`, to: fixed(testSerial)},
	{kinds: kString, key: `(?i)gender`, to: fixed(testGender)},
	{kinds: kString, key: `(?i)time_?zone`, to: fixed(testTimeZone)},
	{kinds: kString, value: timeZoneValue, to: fixed(testTimeZone)},
	{kinds: kString, key: idName, to: fixed(testID)},
	{kinds: kString, value: dateOrTime, to: fixedDate},
	{kinds: kString, jsonPath: `exercises\[\]\.name$`, value: enumWord, to: keep},
	{kinds: kString, key: freeTextName, to: fixed(testText)},
	{kinds: kString, key: `^productDisplayName$`, value: `^[A-Za-z0-9 +-]+$`, to: keep},
	{kinds: kString, key: `(?i)(partNumber|version)$`, value: `^[A-Za-z0-9.-]+$`, to: keep},
	{kinds: kString, value: `^(true|false)$`, to: keep},
	{kinds: kString, key: enumName, value: enumWord, to: keep},
}

func keep(v any) any { return v }

func fixed(s string) func(any) any { return func(any) any { return s } }

func fixedNumber(v any) any {
	if strings.ContainsAny(string(v.(json.Number)), ".eE") {
		return json.Number("0.5")
	}
	return json.Number("1")
}

var digits = regexp.MustCompile(`\d`)

func fixedDate(v any) any {
	s := v.(string)
	if len(s) >= 10 && s[4] == '-' {
		return "2026-01-01" + digits.ReplaceAllString(s[10:], "0")
	}
	return digits.ReplaceAllString(s, "0")
}

type compiledRule struct {
	kinds                kind
	key, jsonPath, value *regexp.Regexp
	to                   func(any) any
}

var compiledRules = func() []compiledRule {
	compile := func(expr string) *regexp.Regexp {
		if expr == "" {
			return nil
		}
		return regexp.MustCompile(expr)
	}
	out := make([]compiledRule, len(scrubRules))
	for i, r := range scrubRules {
		out[i] = compiledRule{r.kinds, compile(r.key), compile(r.jsonPath), compile(r.value), r.to}
	}
	return out
}()

func (r compiledRule) matches(k kind, key, jsonPath string, v any) bool {
	if r.kinds&k == 0 || (r.key != nil && !r.key.MatchString(key)) || (r.jsonPath != nil && !r.jsonPath.MatchString(jsonPath)) {
		return false
	}
	if r.value == nil {
		return true
	}
	s, ok := v.(string)
	return ok && r.value.MatchString(s)
}

func kindOf(v any) kind {
	switch v.(type) {
	case string:
		return kString
	case json.Number:
		return kNumber
	case bool:
		return kBool
	case nil:
		return kNull
	case map[string]any:
		return kObject
	default:
		return kArray
	}
}

// An UnclassifiedError names a value that no rule classifies. It never holds
// the value, which can be personal data.
type UnclassifiedError struct {
	Route string
	Path  string
	Kind  string
}

func (e *UnclassifiedError) Error() string {
	return fmt.Sprintf("%s: %s: no rule in scrubRules classifies this %s", e.Route, e.Path, e.Kind)
}

type scrubber struct {
	route string
	errs  []error
}

func (s *scrubber) value(key, path string, v any) any {
	k := kindOf(v)
	for _, r := range compiledRules {
		if r.matches(k, key, path, v) {
			return r.to(v)
		}
	}
	switch v := v.(type) {
	case map[string]any:
		for name, x := range v {
			v[name] = s.value(name, joinPath(path, name), x)
		}
		return v
	case []any:
		for i, x := range v {
			v[i] = s.value(key, path+"[]", x)
		}
		return v
	}
	s.errs = append(s.errs, &UnclassifiedError{Route: s.route, Path: cmp.Or(path, "(body)"), Kind: k.String()})
	return v
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func (s *scrubber) text(key, path, raw string) string {
	var v any = raw
	if raw != "" && strings.Trim(raw, "0123456789") == "" {
		v = json.Number(raw)
	}
	return fmt.Sprint(s.value(key, path, v))
}

// Scrub replaces the personal data in r with fixed test values. It fails
// when the host is not a Garmin API host, when the path matches no template
// in scrubPaths, or when a value matches no rule in scrubRules. The error
// names each such value by its route and JSON path, never by its content.
func Scrub(r Record) (Record, error) {
	if !slices.Contains(garminHosts, r.Host) {
		return Record{}, fmt.Errorf("%s %s: not a Garmin API host", r.Method, r.Host)
	}
	u, err := url.ParseRequestURI(r.URI)
	if err != nil {
		return Record{}, fmt.Errorf("%s: the URI does not parse", r.Method)
	}
	tmpl, ok := matchTemplate(u.Path)
	if !ok {
		return Record{}, fmt.Errorf("%s %s: no template in scrubPaths matches this path", r.Method, maskPath(u.Path))
	}
	s := &scrubber{route: r.Method + " " + tmpl}

	have, want := strings.Split(u.Path, "/"), strings.Split(tmpl, "/")
	for i, seg := range want {
		if name, ok := placeholder(seg); ok {
			have[i] = url.PathEscape(s.text(name, seg, have[i]))
		}
	}
	uri := strings.Join(have, "/")
	if u.RawQuery != "" {
		var pairs []string
		for _, pair := range strings.Split(u.RawQuery, "&") {
			k, v, _ := strings.Cut(pair, "=")
			key, err1 := url.QueryUnescape(k)
			val, err2 := url.QueryUnescape(v)
			if err1 != nil || err2 != nil {
				return Record{}, fmt.Errorf("%s: the query does not parse", s.route)
			}
			pairs = append(pairs, url.QueryEscape(key)+"="+url.QueryEscape(s.text(key, "?"+key, val)))
		}
		uri += "?" + strings.Join(pairs, "&")
	}

	out := Record{Method: r.Method, Host: r.Host, URI: uri, Status: r.Status, NonJSONBytes: r.NonJSONBytes}
	if r.Body != nil {
		dec := json.NewDecoder(bytes.NewReader(r.Body))
		dec.UseNumber()
		var body any
		if err := dec.Decode(&body); err != nil {
			return Record{}, fmt.Errorf("%s: the body is not JSON", s.route)
		}
		body = s.value("", "", body)
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return Record{}, err
		}
		out.Body = bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
	}
	if len(s.errs) > 0 {
		return Record{}, errors.Join(s.errs...)
	}
	return out, nil
}

func placeholder(seg string) (string, bool) {
	name, ok := strings.CutPrefix(seg, "{")
	if !ok {
		return "", false
	}
	return strings.CutSuffix(name, "}")
}

func matchTemplate(path string) (string, bool) {
	segs := strings.Split(path, "/")
	best, bestLiterals := "", -1
	for _, tmpl := range scrubPaths {
		want := strings.Split(tmpl, "/")
		if len(want) != len(segs) {
			continue
		}
		literals := 0
		for i, w := range want {
			if _, ok := placeholder(w); ok {
				continue
			}
			if w != segs[i] {
				literals = -1
				break
			}
			literals++
		}
		if literals > bestLiterals {
			best, bestLiterals = tmpl, literals
		}
	}
	return best, bestLiterals >= 0
}

var templateLiterals = func() map[string]bool {
	m := map[string]bool{}
	for _, tmpl := range scrubPaths {
		for _, seg := range strings.Split(tmpl, "/") {
			if _, ok := placeholder(seg); !ok {
				m[seg] = true
			}
		}
	}
	return m
}()

func maskPath(path string) string {
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		if !templateLiterals[seg] {
			segs[i] = "*"
		}
	}
	return strings.Join(segs, "/")
}

// ScrubDir scrubs every record file in in and writes the result to out,
// which must not exist. Records that are the same after the scrub are
// written once. When any record fails, ScrubDir returns every failure and
// writes nothing.
func ScrubDir(in, out string) error {
	paths, err := filepath.Glob(filepath.Join(in, "*.json"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("%s holds no record files", in)
	}
	files := map[string][]byte{}
	failed := map[string][]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rec, err := DecodeRecord(data)
		if err == nil {
			rec, err = Scrub(rec)
		}
		if err != nil {
			errs := []error{err}
			if joined, ok := err.(interface{ Unwrap() []error }); ok {
				errs = joined.Unwrap()
			}
			for _, e := range errs {
				failed[e.Error()] = append(failed[e.Error()], filepath.Base(path))
			}
			continue
		}
		data, err = EncodeRecord(rec)
		if err != nil {
			return err
		}
		files[recordName(rec, data)] = data
	}
	if len(failed) > 0 {
		var lines []string
		for _, msg := range slices.Sorted(maps.Keys(failed)) {
			lines = append(lines, fmt.Sprintf("%s (%d records, first %s)", msg, len(failed[msg]), failed[msg][0]))
		}
		return fmt.Errorf("scrub failed, nothing written:\n%s", strings.Join(lines, "\n"))
	}
	if err := os.Mkdir(out, 0o755); err != nil {
		return err
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(out, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9.-]+`)

func recordName(r Record, data []byte) string {
	path, _, _ := strings.Cut(strings.TrimPrefix(r.URI, "/"), "?")
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%s_%s_%d_%s.json", strings.ToLower(r.Method),
		unsafeNameChars.ReplaceAllString(strings.ReplaceAll(path, "/", "."), "-"), r.Status, hex.EncodeToString(sum[:4]))
}
