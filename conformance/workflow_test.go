package conformance

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// The scheduled e2e job records real responses from the owner's account.
// The scrubbed set keeps the keys and the shape of each response, so it
// stays on the runner: the job must not upload it, and must not be able to
// push it.
func TestE2EWorkflowKeepsTheRecordedSetOnTheRunner(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/e2e.yml")
	if err != nil {
		t.Fatal(err)
	}
	yml := string(data)
	for _, banned := range []string{"upload-artifact", "upload-pages-artifact", "git push", "gh pr", "gh release"} {
		if strings.Contains(yml, banned) {
			t.Errorf("e2e.yml uses %q: the recorded set must not leave the runner", banned)
		}
	}
	if !strings.Contains(yml, "\npermissions:\n  contents: read\n\n") {
		t.Error("e2e.yml must grant the workflow contents: read and nothing else")
	}
}

// go test prints the log of a passing test only with -v. The replay tests
// log a route whose elements no recording lets them check, and a case
// they do not run. Without -v the scheduled job shows neither, and a
// route that is always empty on the account passes with no signal.
func TestE2EWorkflowReplaysVerbose(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/e2e.yml")
	if err != nil {
		t.Fatal(err)
	}
	var replays []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "go test") && strings.Contains(line, "TestRecorded") {
			replays = append(replays, line)
		}
	}
	if len(replays) != 1 {
		t.Fatalf("e2e.yml has %d go test lines that run TestRecorded, want 1", len(replays))
	}
	if !slices.Contains(strings.Fields(replays[0]), "-v") {
		t.Errorf("the replay step %q runs go test without -v, so the job log never shows the routes the tests leave unchecked", strings.TrimSpace(replays[0]))
	}
}
