package conformance

import (
	"bytes"
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

// goldenName is the display name in the golden routes, so a scrubbed path
// still matches them. fixedDigits keeps a path segment numeric for replayKey.
const (
	goldenName  = "tester"
	fixedDigits = "1"
	fixedDate   = "2026-01-01"
	fixedBool   = true
)

var (
	digitsOnly = regexp.MustCompile(`^\d+$`)
	dateOnly   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	dateOrTime = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}([T ]\d{2}:\d{2}(:\d{2}(\.\d+)?)?)?|\d{2}:\d{2}(:\d{2})?)(Z|[+-]\d{2}:?\d{2})?$`)
	digit      = regexp.MustCompile(`\d`)
	// Garmin keys are lower camelCase, or UPPER_SNAKE in maps keyed by an
	// enum such as metricsMap.
	identifierKey = regexp.MustCompile(`^([a-z][A-Za-z0-9]*|[A-Z][A-Z0-9_]*)$`)
)

func fixedString(s string) string {
	switch {
	case digitsOnly.MatchString(s):
		return fixedDigits
	case dateOrTime.MatchString(s):
		if len(s) >= 10 && s[4] == '-' {
			return fixedDate + digit.ReplaceAllString(s[10:], "0")
		}
		return digit.ReplaceAllString(s, "0")
	}
	return goldenName
}

func fixedNumber(n json.Number) json.Number {
	if strings.ContainsAny(string(n), ".eE") {
		return "0.5"
	}
	return "1"
}

// A KeyError names an object key that is neither an identifier nor a date.
// Path holds only structure: `$` is the body, `[]` an element of an array
// and `<key i>` the i-th key of an object in byte order from 0. The key
// itself can be personal data and is never in the error.
type KeyError struct {
	Route string
	Path  string
}

func (e *KeyError) Error() string {
	return fmt.Sprintf("%s: %s: the key is not an identifier or a date", e.Route, e.Path)
}

type scrubber struct {
	route string
	errs  []error
}

// fail records a bad key once per position: every element of a series
// has the same position.
func (s *scrubber) fail(at string) {
	if !slices.ContainsFunc(s.errs, func(e error) bool { return e.(*KeyError).Path == at }) {
		s.errs = append(s.errs, &KeyError{Route: s.route, Path: at})
	}
}

func (s *scrubber) value(path string, v any) any {
	switch v := v.(type) {
	case string:
		return fixedString(v)
	case json.Number:
		return fixedNumber(v)
	case bool:
		return fixedBool
	case nil:
		return nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for i, key := range slices.Sorted(maps.Keys(v)) {
			at := fmt.Sprintf("%s.<key %d>", path, i)
			x := s.value(at, v[key])
			switch {
			case identifierKey.MatchString(key):
				out[key] = x
			case dateOnly.MatchString(key):
				out[fixedDate] = x
			default:
				s.fail(at)
			}
		}
		return out
	}
	arr := v.([]any)
	if len(arr) == 0 {
		return arr
	}
	var merged any
	for _, x := range arr {
		merged = merge(merged, s.value(path+"[]", x))
	}
	return []any{merged}
}

// merge joins two scrubbed values into one that holds every key and the
// element of both, so which element of a series has a key, a value or a
// null does not survive. A null gives way to a value. When the kinds
// differ, a wins.
func merge(a, b any) any {
	switch a := a.(type) {
	case nil:
		return b
	case map[string]any:
		if b, ok := b.(map[string]any); ok {
			for k, v := range b {
				a[k] = merge(a[k], v)
			}
		}
	case []any:
		if b, ok := b.([]any); ok && len(b) > 0 {
			if len(a) == 0 {
				return b
			}
			a[0] = merge(a[0], b[0])
		}
	}
	return a
}

// Scrub replaces every value in r with a fixed one and collapses every
// array to one element that holds every key of its elements and a value
// where any element has one, so neither a value, nor the length of a
// series, nor the position of a null or of a key in it survives. An empty
// array stays empty and an all-null array becomes [null]. Scrub fails when the
// host is not a Garmin API host, when the path matches no template in
// scrubPaths, or when an object or query key is neither an identifier nor
// a date. The error names the route and the position, never a key or a
// value.
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

	have := strings.Split(u.Path, "/")
	for i, seg := range strings.Split(tmpl, "/") {
		if isPlaceholder(seg) {
			have[i] = url.PathEscape(fixedString(have[i]))
		}
	}
	uri := strings.Join(have, "/")
	if u.RawQuery != "" {
		var pairs []string
		for i, pair := range strings.Split(u.RawQuery, "&") {
			k, v, _ := strings.Cut(pair, "=")
			key, err1 := url.QueryUnescape(k)
			val, err2 := url.QueryUnescape(v)
			if err1 != nil || err2 != nil {
				return Record{}, fmt.Errorf("%s: the query does not parse", s.route)
			}
			if !identifierKey.MatchString(key) {
				s.errs = append(s.errs, &KeyError{Route: s.route, Path: fmt.Sprintf("?<key %d>", i)})
			}
			pairs = append(pairs, url.QueryEscape(key)+"="+url.QueryEscape(fixedString(val)))
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
		body = s.value("$", body)
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

func isPlaceholder(seg string) bool {
	return strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")
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
			if isPlaceholder(w) {
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
			if !isPlaceholder(seg) {
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
