package conformance

import (
	"os"
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
