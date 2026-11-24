package main

import (
	"encoding/json"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// The release build legs live in a script rather than in the workflow because a
// narrower release expands a subset of the same six, and a table written twice
// is one that drifts. These tests read them the way the workflow does.
func releaseBuildLegs(t *testing.T, platforms string) []map[string]string {
	t.Helper()
	cmd := exec.Command("bash", "scripts/studio-build-matrix.sh", platforms)
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("studio-build-matrix.sh %s: %v: %s", platforms, err, out)
	}
	var matrix struct {
		Include []map[string]string `json:"include"`
	}
	if err := json.Unmarshal(out, &matrix); err != nil {
		t.Fatalf("studio-build-matrix.sh %s printed %q, which is not the matrix JSON", platforms, out)
	}
	return matrix.Include
}

// A release that claims only Windows expands two legs and the platforms it does
// not claim have none, so nothing is queued on a runner that may never answer.
func TestWindowsOnlyReleaseExpandsOnlyTheWindowsLegs(t *testing.T) {
	legs := releaseBuildLegs(t, "windows")
	if len(legs) != 2 {
		t.Fatalf("a Windows-only release expands %d legs, want 2", len(legs))
	}
	for _, leg := range legs {
		if !strings.HasPrefix(leg["platform"], "windows/") {
			t.Errorf("a Windows-only release expands %s", leg["platform"])
		}
	}
	root := parseWorkflowFile(t, "../../.github/workflows/release-studio.yml")
	jobs := jobsOf(t, root)
	for _, arch := range []string{"amd64", "arm64"} {
		artifact := "studio-build-windows-" + arch
		downloaded := false
		for _, step := range mappingValue(mappingValue(jobs, "publish"), "steps").Content {
			if mappingScalar(mappingValue(step, "with"), "name") == artifact {
				downloaded = true
			}
		}
		if !downloaded {
			t.Errorf("publish never downloads %s", artifact)
		}
	}
}

// The full line is the six the runbook names, and every one of them is reachable
// by the set a tag push builds.
func TestEveryPlatformHasALegInTheFullSet(t *testing.T) {
	want := []string{
		"windows/amd64", "windows/arm64",
		"darwin/amd64", "darwin/arm64",
		"linux/amd64", "linux/arm64",
	}
	var got []string
	for _, leg := range releaseBuildLegs(t, "all") {
		got = append(got, leg["platform"])
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the full set expands %v, want %v", got, want)
	}
}
