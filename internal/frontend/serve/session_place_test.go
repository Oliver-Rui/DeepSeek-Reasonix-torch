package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/state/sessionstore"
)

func placeSessionRequest(t *testing.T, srv *httptest.Server, path, before string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"path": path, "before": before})
	if err != nil {
		t.Fatal(err)
	}
	return postLaunchJSON(t, srv.URL+"/tree/sessions/move", string(body), nil)
}

func sessionNames(rows []sessionstore.SessionInfo) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = strings.TrimSuffix(filepath.Base(row.Path), ".jsonl")
	}
	return out
}

func TestPlacingASessionMovesItBeforeTheNamedTarget(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	paths := seedAged(t, root, 4)
	// Activity order is session-003, 002, 001, 000.
	if status, code := placeSessionRequest(t, srv, paths[0], paths[3]); status != http.StatusNoContent {
		t.Fatalf("place = %d %s", status, code)
	}
	if got, want := rowNames(treeFor(t, srv, "")), []string{"session-000", "session-003", "session-002", "session-001"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if status, code := placeSessionRequest(t, srv, paths[1], paths[3]); status != http.StatusNoContent {
		t.Fatalf("second place = %d %s", status, code)
	}
	if got, want := rowNames(treeFor(t, srv, "")), []string{"session-000", "session-001", "session-003", "session-002"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	// Before the row it already precedes is the same place: no write, no error.
	if status, code := placeSessionRequest(t, srv, paths[1], paths[3]); status != http.StatusNoContent {
		t.Fatalf("no-op place = %d %s", status, code)
	}
	if got := rowNames(treeFor(t, srv, "")); !slices.Equal(got, []string{"session-000", "session-001", "session-003", "session-002"}) {
		t.Fatalf("a no-op changed the order: %v", got)
	}
}

func TestPlacingASessionRefusesThePinnedAndArchivedBoundaries(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	paths := seedAged(t, root, 4)
	if err := sessionstore.SetSessionPinned(paths[3], true); err != nil {
		t.Fatal(err)
	}
	if err := sessionstore.SetSessionLineageArchived(paths[2], true); err != nil {
		t.Fatal(err)
	}
	if status, code := placeSessionRequest(t, srv, paths[0], paths[3]); status != http.StatusConflict || code != codeSessionOrderBoundary {
		t.Fatalf("crossing the pin boundary = %d %s, want 409 %s", status, code, codeSessionOrderBoundary)
	}
	if status, code := placeSessionRequest(t, srv, paths[0], paths[2]); status != http.StatusConflict || code != codeSessionOrderBoundary {
		t.Fatalf("crossing the archive boundary = %d %s, want 409 %s", status, code, codeSessionOrderBoundary)
	}
	// Inside the pinned partition the place is a normal order write.
	if err := sessionstore.SetSessionPinned(paths[1], true); err != nil {
		t.Fatal(err)
	}
	if status, code := placeSessionRequest(t, srv, paths[1], paths[3]); status != http.StatusNoContent {
		t.Fatalf("pinned place = %d %s", status, code)
	}
	listed, _ := h.treeSessionRowsForProject([]string{root})
	pinned := []sessionstore.SessionInfo{}
	for _, row := range listed {
		if row.Pinned {
			pinned = append(pinned, row)
		}
	}
	if got, want := sessionNames(pinned), []string{"session-001", "session-003"}; !slices.Equal(got, want) {
		t.Fatalf("pinned order = %v, want %v", got, want)
	}
}

func TestPlacingASessionSpansAWorktreeProjectsRoots(t *testing.T) {
	root, copy := projectWithWorktree(t)
	rememberWorkspace(root)
	rememberWorkspace(copy)
	local := seedAged(t, root, 2)
	isolated := seedAged(t, copy, 2)
	h := NewHub(HubOptions{})
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	moved, target := isolated[0], local[1]
	if status, code := placeSessionRequest(t, srv, moved, target); status != http.StatusNoContent {
		t.Fatalf("worktree place = %d %s", status, code)
	}
	rows := treeFor(t, srv, "?limit=0").Sessions
	from := slices.IndexFunc(rows, func(row treeSession) bool { return row.Path == moved })
	to := slices.IndexFunc(rows, func(row treeSession) bool { return row.Path == target })
	if from < 0 || to != from+1 {
		t.Fatalf("moved row at %d, target at %d; want the moved one immediately before the target", from, to)
	}
}
