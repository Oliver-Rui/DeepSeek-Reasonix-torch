package sessionstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
)

// A dispatched session's first (and only) turns arrive from another session.
// They are host-authored, so the sidecar counts zero human turns while the
// transcript holds real content. The meta records the peer content, and the
// sidebar keeps the row; a session with no such mark and no turns stays hidden.
func TestListSessionsKeepsContentWithNoHumanTurns(t *testing.T) {
	dir := testenv.TempDir(t)
	used := filepath.Join(dir, "20260101-000000.000000000-group-glm.jsonl")
	empty := filepath.Join(dir, "20260101-000001.000000000-group-glm.jsonl")
	if err := os.WriteFile(used, []byte(`{"role":"user","content":"<peer-message from=\"a\">hi</peer-message>"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveBranchMetaPreserveUpdated(used, BranchMeta{SchemaVersion: BranchMetaCountsVersion, Turns: 0, PeerContent: true, UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := SaveBranchMetaPreserveUpdated(empty, BranchMeta{SchemaVersion: BranchMetaCountsVersion, Turns: 0, UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	rows, err := ListSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, row := range rows {
		found[filepath.Base(row.Path)] = true
	}
	if !found[filepath.Base(used)] {
		t.Fatalf("a content-bearing session with no human turns is missing: %v", found)
	}
	if found[filepath.Base(empty)] {
		t.Fatalf("an empty session is listed: %v", found)
	}
}
