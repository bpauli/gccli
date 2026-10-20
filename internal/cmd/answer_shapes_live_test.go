//go:build answershapes

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
)

// liveSkip names the reads that the live refresh never sends, with the
// reason. Each keeps its registry line.
var liveSkip = map[string]string{
	"reload":         "the GET asks Garmin to reload data",
	"exercises list": "the catalog is a public file that fetchExerciseCatalogFn reads outside the client",
	"devices alarms": "gccli builds the list from the alarms in the settings of each device",
}

// TestAnswerShapesLive refreshes the answer shape registry from the raw
// Garmin answers. For every read in the registry's scope it runs the
// command in-process with --json and the args of the main golden (or
// accountArgs), with a client whose transport records the status and body
// of the last answer and refuses any request that is not a GET. The shape
// comes from bodyShape over that body, never from what gccli printed: an
// empty body is no evidence, so it keeps a line that has a shape, writes
// unknown for a line without one, and fails on a line whose source claims
// live evidence that the refresh cannot see. A read that exits with an
// error keeps its line. The test prints every registry line, fails on a
// line that differs from the file, and with -update writes the file.
//
// ANSWER_SHAPES_CONFIG names the XDG_CONFIG_HOME of the stored login of
// the test account, with its file keyring. IsolatedMain strips every
// GCCLI_* variable and HOME, so the test sets XDG_CONFIG_HOME and the
// file backend itself. The client never refreshes the token, so an
// expired login fails every read:
//
//	ANSWER_SHAPES_CONFIG=/path/to/.config go test -tags answershapes ./internal/cmd -run TestAnswerShapesLive -update
func TestAnswerShapesLive(t *testing.T) {
	dir := os.Getenv("ANSWER_SHAPES_CONFIG")
	if dir == "" {
		t.Skip("set ANSWER_SHAPES_CONFIG to the XDG_CONFIG_HOME of the stored login of the test account")
	}
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("GCCLI_KEYRING_BACKEND", "file")
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account() == "" {
		t.Fatalf("%s has no default account", dir)
	}
	t.Logf("account %s", cfg.Account())

	reads := answerReads(t)
	for path := range liveSkip {
		if _, ok := reads[path]; !ok {
			t.Fatalf("liveSkip names %q, which is not a read that prints the Garmin answer: remove the entry", path)
		}
	}
	for _, path := range slices.Sorted(maps.Keys(reads)) {
		if r := reads[path]; !r.custom {
			e, ok := garminapi.Describe(r.leaf.Endpoint)
			if !ok || e.Method != http.MethodGet || e.Body {
				t.Fatalf("%s: endpoint %s is not a GET without a body: the live refresh sends only reads", path, r.leaf.Endpoint)
			}
		}
	}

	rec := &answerRecorder{}
	stub(t, &newClientFn, func(tokens *garminauth.Tokens) *garminapi.Client {
		return garminapi.NewClient(tokens, garminapi.WithAttemptTransport(rec.wrap))
	})
	stub(t, &refreshFn, func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, errors.New("the live refresh does not refresh the token")
	})

	shapes := readAnswerShapes(t)
	today := time.Now().Format("2006-01-02")
	for _, path := range slices.Sorted(maps.Keys(reads)) {
		if why, ok := liveSkip[path]; ok {
			t.Logf("%s: not sent: %s", path, why)
			continue
		}
		args, err := liveArgs(path)
		if err != nil {
			t.Error(err)
			continue
		}
		rec.reset()
		var stdout, stderr bytes.Buffer
		code := execute(append([]string{"--json", "--color", "never"}, args...), "test", "none", "never", &stdout, &stderr)
		last, refused := rec.result()
		switch {
		case refused != "":
			t.Errorf("%s: sent %s, which is not a GET: the live refresh sends only reads", path, refused)
			continue
		case code != 0:
			t.Logf("%s: exit %d (%s): keeps its line", path, code, firstLine(stderr.String()))
			continue
		case last == nil:
			t.Errorf("%s: exit 0 without a request through the client", path)
			continue
		}
		answered := fmt.Sprintf("%s %s answered %d", last.method, last.path, last.status)
		switch shape := bodyShape(last.status, last.body); shape {
		case "array", "object":
			t.Logf("%s: %s with an %s", path, answered, shape)
			shapes[path] = answerShape{Shape: shape, Source: "live " + today}
		case "empty":
			s, ok := shapes[path]
			switch {
			case !ok || s.Shape == "unknown":
				t.Logf("%s: %s with an empty body: unknown", path, answered)
				shapes[path] = answerShape{Shape: "unknown", Source: "empty body live " + today}
			case strings.HasPrefix(s.Source, "live "):
				t.Errorf("%s: %s with an empty body, but the line says %s (%s), and the refresh cannot see that evidence: add args that answer to accountArgs, or set the line to unknown with the source \"empty body live <date>\"", path, answered, s.Shape, s.Source)
			default:
				t.Logf("%s: %s with an empty body: keeps its line (%s, %s)", path, answered, s.Shape, s.Source)
			}
		default:
			t.Errorf("%s: %s with a body that is not an array or an object: %.60q", path, answered, last.body)
		}
	}
	data := formatAnswerShapes(shapes)
	os.Stdout.Write(data)
	if *update {
		if err := os.WriteFile(answerShapesFile, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	current, err := os.ReadFile(answerShapesFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, data) {
		t.Errorf("%s differs from the live run (- file, + live):\n%s", answerShapesFile, lineDiff(string(current), string(data)))
	}
}

func liveArgs(path string) ([]string, error) {
	if args, ok := accountArgs[path]; ok {
		return args, nil
	}
	g, err := readGolden(goldenFile(path))
	if err != nil {
		return nil, err
	}
	args, _, err := g.input()
	return args, err
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

type recordedAnswer struct {
	method, path string
	status       int
	body         []byte
}

// answerRecorder keeps the last answer that passed through the client, and
// the first request that it refused because it was not a GET.
type answerRecorder struct {
	mu      sync.Mutex
	last    *recordedAnswer
	refused string
}

func (r *answerRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.last, r.refused = nil, ""
}

func (r *answerRecorder) result() (*recordedAnswer, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last, r.refused
}

func (r *answerRecorder) wrap(next http.RoundTripper) http.RoundTripper {
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			r.mu.Lock()
			if r.refused == "" {
				r.refused = req.Method + " " + req.URL.RequestURI()
			}
			r.mu.Unlock()
			return nil, fmt.Errorf("the live refresh sends only GET requests, not %s %s", req.Method, req.URL.RequestURI())
		}
		resp, err := next.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		r.mu.Lock()
		r.last = &recordedAnswer{method: req.Method, path: req.URL.RequestURI(), status: resp.StatusCode, body: body}
		r.mu.Unlock()
		return resp, nil
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
