//go:build answershapes

package cmd

import (
	"bytes"
	"errors"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// liveSkip names the JSON rows that the live refresh never sends, with the
// reason. Each keeps its registry line.
var liveSkip = map[string]string{
	"reload": "the GET asks Garmin to reload data",
}

// TestAnswerShapesLive refreshes the answer shape registry from Garmin. It
// runs the command named by ANSWER_SHAPES_BIN with --json and the args of
// the main golden (or accountArgs) for every Leaf row with a JSON output,
// read-only: a row whose endpoint is not a GET without a body fails the
// test before anything is sent. A row that answers 200 gets its shape from
// the first byte of stdout and the source "live <today>". A row without a
// 200 keeps its line. The test prints every registry line, fails on a line
// that differs from the file, and with -update writes the file.
//
// IsolatedMain strips every GCCLI_* variable and HOME from the test, so
// ANSWER_SHAPES_BIN names a wrapper script that sets HOME, XDG_CONFIG_HOME
// and GCCLI_KEYRING_BACKEND for the login of the test account and then runs
// the gccli binary with its arguments:
//
//	ANSWER_SHAPES_BIN=/path/to/wrapper go test -tags answershapes ./internal/cmd -run TestAnswerShapesLive -update
func TestAnswerShapesLive(t *testing.T) {
	bin := os.Getenv("ANSWER_SHAPES_BIN")
	if bin == "" {
		t.Skip("set ANSWER_SHAPES_BIN to a wrapper that runs gccli with the login of the test account")
	}
	for path := range liveSkip {
		if _, ok := jsonLeaves()[path]; !ok {
			t.Fatalf("liveSkip names %q, which is not a Leaf row with a JSON output: remove the entry", path)
		}
	}
	shapes := readAnswerShapes(t)
	leaves := jsonLeaves()
	for _, path := range slices.Sorted(maps.Keys(leaves)) {
		e, ok := garminapi.Describe(leaves[path].Endpoint)
		if !ok || e.Method != http.MethodGet || e.Body {
			t.Fatalf("%s: endpoint %s is not a GET without a body: the live refresh sends only reads", path, leaves[path].Endpoint)
		}
	}
	today := time.Now().Format("2006-01-02")
	for _, path := range slices.Sorted(maps.Keys(leaves)) {
		if why, ok := liveSkip[path]; ok {
			t.Logf("%s: not sent: %s", path, why)
			continue
		}
		args, err := liveArgs(path)
		if err != nil {
			t.Error(err)
			continue
		}
		shape, note := liveShape(bin, args)
		if shape == "" {
			t.Logf("%s: no 200 (%s): keeps its line", path, note)
			continue
		}
		shapes[path] = answerShape{Shape: shape, Source: "live " + today}
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

// liveShape runs the binary with --json and args. It returns array or
// object for a 200 answer, or "" with a note for any other result.
func liveShape(bin string, args []string) (shape, note string) {
	cmd := exec.Command(bin, append([]string{"--json"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return "", strings.TrimSpace(stderr.String())
	case err != nil:
		return "", err.Error()
	}
	switch out := bytes.TrimSpace(stdout.Bytes()); {
	case bytes.HasPrefix(out, []byte("[")):
		return "array", ""
	case bytes.HasPrefix(out, []byte("{")):
		return "object", ""
	}
	return "", "stdout is not JSON"
}
