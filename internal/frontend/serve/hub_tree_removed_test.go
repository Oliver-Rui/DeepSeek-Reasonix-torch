package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/config"
)

func postRemoveWorkspace(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	body, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(srv.URL+"/tree/workspaces/remove", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func treeRoots(t *testing.T, srv *httptest.Server) []treeWorkspace {
	t.Helper()
	return hubGet[[]treeWorkspace](t, srv, "/tree")
}

// treeRootsArchived reads the tree the way the sidebar's 已归档 scope does:
// that is where a removed project's ghost row stands.
func treeRootsArchived(t *testing.T, srv *httptest.Server) []treeWorkspace {
	t.Helper()
	return hubGet[[]treeWorkspace](t, srv, "/tree?scope=archived")
}

func treeRowFor(rows []treeWorkspace, dir string) *treeWorkspace {
	for i := range rows {
		if sameFolder(rows[i].Root, dir) {
			return &rows[i]
		}
	}
	return nil
}

// The pane a launch opens in a folder nobody picked lists that folder once
// something was said there. The user removing the row is the last word about
// listing it: the removal is recorded, no later launch lists the folder whole
// again, nothing on disk is touched. What removal leaves is the ghost row, and
// the ghost row stands where the archive is the view — the 已归档 scope.
func TestARemovedFolderStaysRemovedAcrossALaunch(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	implicit := testenv.TempDir(t)
	writeSessionAt(t, filepath.Join(SessionDirFor(implicit), "held.jsonl"))
	if ensureAutoWorkspace() == "" {
		t.Fatal("the auto workspace could not be created")
	}

	hub := NewHub(HubOptions{})
	defer hub.Shutdown()
	hubRuntime(t, hub, implicit) // the pane a launch opens there
	srv := httptest.NewServer(operatorHandler(hub))
	defer srv.Close()

	if row := treeRowFor(treeRoots(t, srv), implicit); row == nil {
		t.Fatal("a folder a conversation was held in showed no row to open it from")
	}
	resp := postRemoveWorkspace(t, srv, implicit)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove = %d, want 204", resp.StatusCode)
	}
	// The pane was released by the removal, so the folder is archive-only: the
	// everyday list is clean, the 已归档 scope carries the ghost.
	if treeRowFor(treeRoots(t, srv), implicit) != nil {
		t.Fatalf("an archive-only ghost row leaked into the everyday list: %+v", treeRoots(t, srv))
	}
	ghost := treeRowFor(treeRootsArchived(t, srv), implicit)
	if ghost == nil || !ghost.Removed || !ghost.Remembered {
		t.Fatalf("the 已归档 scope shows no ghost row for the archive: %+v", treeRootsArchived(t, srv))
	}

	// The removal outlives the session that recorded it.
	list, err := readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(list.Removed, projectIdentity(implicit)) {
		t.Fatalf("no removal on record: %+v", list.Removed)
	}
	if slices.Contains(list.Paths, implicit) {
		t.Error("a removed folder stayed in the remembered list")
	}
	if _, err := os.Stat(filepath.Join(SessionDirFor(implicit), "held.jsonl")); err != nil {
		t.Errorf("the removal touched the conversation on disk: %v", err)
	}

	// The next launch opens a pane there again; live work keeps the row in the
	// everyday list, but the folder is still not listed whole — the row is the
	// ghost's, showing the archive and the pane's own conversation.
	hub2 := NewHub(HubOptions{})
	defer hub2.Shutdown()
	hubRuntime(t, hub2, implicit)
	srv2 := httptest.NewServer(operatorHandler(hub2))
	defer srv2.Close()
	rows := treeRoots(t, srv2)
	if ghost := treeRowFor(rows, implicit); ghost == nil || !ghost.Removed {
		t.Fatalf("a ghost row with live work left the everyday list: %+v", rows)
	}
	if !slices.Contains(readListRemoved(t), projectIdentity(implicit)) {
		t.Error("a launch lifted the removal on record")
	}
	// The 任务 group stands, as does a project the user picked.
	if row := treeRowFor(rows, config.AutoWorkspaceDir()); row == nil {
		t.Fatal("the auto workspace lost its standing row")
	}
	remembered := testenv.TempDir(t)
	if err := addRememberedWorkspace(t.Context(), remembered); err != nil {
		t.Fatal(err)
	}
	if row := treeRowFor(treeRoots(t, srv2), remembered); row == nil || !row.Remembered {
		t.Fatalf("a remembered project did not list as one: %+v", treeRoots(t, srv2))
	}
}

func readListRemoved(t *testing.T) []string {
	t.Helper()
	list, err := readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	return list.Removed
}

// Asking for the folder again — the add the sidebar offers — is what lifts a
// removal: the row returns and the removal is no longer on record.
func TestAskingForARemovedFolderListsItAgain(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	dir := testenv.TempDir(t)
	forgetWorkspace(dir)

	list, err := readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(list.Removed, projectIdentity(dir)) {
		t.Fatalf("no removal on record after forget: %+v", list.Removed)
	}
	if slices.Contains(Workspaces(), dir) {
		t.Errorf("a removed folder still reads as remembered")
	}

	if err := addRememberedWorkspace(t.Context(), dir); err != nil {
		t.Fatal(err)
	}
	list, err = readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(list.Removed, projectIdentity(dir)) {
		t.Error("the removal stood over the user asking for the folder back")
	}
	if !slices.Contains(Workspaces(), dir) {
		t.Error("the folder the user asked for does not read as remembered")
	}
}

// A folder a conversation now lives in is listed behind the remembered ones —
// unless the user removed it, which stands over that the way it stands over
// the launch itself.
func TestKeepUsedWorkspaceHonorsARemoval(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	first, removed := testenv.TempDir(t), testenv.TempDir(t)
	if err := addRememberedWorkspace(t.Context(), first); err != nil {
		t.Fatal(err)
	}
	forgetWorkspace(removed)

	keepUsedWorkspace(first)
	keepUsedWorkspace(removed)
	list, err := readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(list.Paths, first) {
		t.Errorf("the folder a conversation lives in was not listed: %+v", list.Paths)
	}
	if slices.Contains(list.Paths, removed) {
		t.Errorf("a removed folder was listed again by a conversation moving in: %+v", list.Paths)
	}
	if !slices.Contains(list.Removed, projectIdentity(removed)) {
		t.Errorf("the removal was dropped from the record: %+v", list.Removed)
	}
}

// The auto workspace is the app's own folder and the standing 任务 group, not
// a project: removing it removes nothing, and its row never loses its place.
func TestForgetDoesNotRecordARemovalOfTheAutoWorkspace(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	if ensureAutoWorkspace() == "" {
		t.Fatal("the auto workspace could not be created")
	}
	auto := config.AutoWorkspaceDir()
	forgetWorkspace(auto)

	list, err := readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Removed) != 0 {
		t.Errorf("the auto workspace was tombstoned: %+v", list.Removed)
	}
	hub := NewHub(HubOptions{})
	defer hub.Shutdown()
	hubRuntime(t, hub, testenv.TempDir(t))
	srv := httptest.NewServer(operatorHandler(hub))
	defer srv.Close()
	if row := treeRowFor(treeRoots(t, srv), auto); row == nil || !row.Auto {
		t.Fatalf("the auto workspace lost its row: %+v", treeRoots(t, srv))
	}
}

// The removal is keyed by identity, so a folder spelled differently — a pane's
// absolute real path against the list's own spelling — still counts as removed.
func TestARemovalMatchesAnySpellingOfTheFolder(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	dir := testenv.TempDir(t)
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	forgetWorkspace(abs)
	recast := filepath.Clean(filepath.Join(abs, ".", "sub", ".."))
	if !workspaceRemoved(recast) {
		t.Fatalf("a removal recorded for %q does not match %q", abs, recast)
	}
}

// A reorder is not a removal: dropping a project before another, or stepping it
// with the menu, rewrites the order and nothing else. The tombstone is what the
// user's "take this off the sidebar" means, and a drag must never write one —
// that mistake would make the row vanish on the next reload.
func TestReorderingProjectsWritesNoRemoval(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	base := testenv.TempDir(t)
	mk := func(name string) string {
		dir := filepath.Join(base, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	a, b, c := mk("alpha"), mk("beta"), mk("gamma")
	if err := updateWorkspaceList(context.Background(), true, func(list *workspaceList) error {
		list.Paths = []string{a, b, c}
		list.Launch = a
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	hub := NewHub(HubOptions{})
	defer hub.Shutdown()
	hubRuntime(t, hub, a)
	srv := httptest.NewServer(operatorHandler(hub))
	defer srv.Close()

	move := func(body string) {
		t.Helper()
		resp, err := http.Post(srv.URL+"/tree/workspaces/move", "application/json", bytes.NewReader([]byte(body)))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("move %s: status %d", body, resp.StatusCode)
		}
	}
	// A drop names its place; a step names a direction. Both are reorders.
	move(`{"path":` + quoteJSON(c) + `,"before":` + quoteJSON(a) + `}`)
	move(`{"path":` + quoteJSON(c) + `,"before":""}`)
	move(`{"path":` + quoteJSON(b) + `,"direction":-1}`)

	list, err := readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Removed) != 0 {
		t.Fatalf("a reorder recorded removals: %+v", list.Removed)
	}
	if len(list.Paths) != 3 {
		t.Fatalf("a reorder lost projects: %+v", list.Paths)
	}
	rows := treeRoots(t, srv)
	for _, dir := range []string{a, b, c} {
		if treeRowFor(rows, dir) == nil {
			t.Fatalf("project %s left the tree: %+v", filepath.Base(dir), rows)
		}
	}
}

// A removed project's archive stays reachable for as long as it exists: the
// ghost row lists the archived conversations, unarchiving one works there the
// same as anywhere, and deleting the last conversation retires the record —
// the ghost row and the tombstone go together, so nothing invisible is kept.
func TestARemovedProjectKeepsItsArchiveUntilItEmpties(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	dir := testenv.TempDir(t)
	sessionPath := filepath.Join(SessionDirFor(dir), "kept.jsonl")
	writeSessionAt(t, sessionPath)
	if err := addRememberedWorkspace(t.Context(), dir); err != nil {
		t.Fatal(err)
	}

	hub := NewHub(HubOptions{})
	defer hub.Shutdown()
	srv := httptest.NewServer(operatorHandler(hub))
	defer srv.Close()

	resp := postRemoveWorkspace(t, srv, dir)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove = %d, want 204", resp.StatusCode)
	}
	ghost := treeRowFor(treeRootsArchived(t, srv), dir)
	if ghost == nil || !ghost.Removed {
		t.Fatalf("the removed project left no ghost row in the 已归档 scope: %+v", treeRootsArchived(t, srv))
	}
	if treeRowFor(treeRoots(t, srv), dir) != nil {
		t.Fatalf("an archive-only ghost row leaked into the everyday list: %+v", treeRoots(t, srv))
	}
	if ghost.Archived != 1 || len(ghost.Sessions) != 1 || !ghost.Sessions[0].Archived {
		t.Fatalf("the ghost row does not show the archived conversation: %+v", ghost)
	}

	// Unarchive is a tree mutation like any other; a removed project's session
	// dir is still owned, so the row answers the same actions it always did.
	post := func(route string, body any) int {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		answer, err := http.Post(srv.URL+route, "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer answer.Body.Close()
		return answer.StatusCode
	}
	if code := post("/tree/sessions/archive", map[string]any{"path": sessionPath, "archived": false}); code != http.StatusNoContent {
		t.Fatalf("unarchive on a ghost row = %d, want 204", code)
	}
	// An unarchived conversation is live work again: the row returns to the
	// everyday list with it.
	ghost = treeRowFor(treeRoots(t, srv), dir)
	if ghost == nil || ghost.Archived != 0 || len(ghost.Sessions) != 1 || ghost.Sessions[0].Archived {
		t.Fatalf("the unarchived conversation is not a live row of the ghost: %+v", ghost)
	}

	// The last conversation goes; the record retires with it.
	if code := post("/tree/sessions/remove", map[string]any{"path": sessionPath}); code != http.StatusNoContent {
		t.Fatalf("remove = %d, want 204", code)
	}
	if rows := treeRoots(t, srv); treeRowFor(rows, dir) != nil {
		t.Fatalf("an emptied ghost row outlived its archive: %+v", rows)
	}
	if removed := readListRemoved(t); slices.Contains(removed, projectIdentity(dir)) {
		t.Errorf("an emptied removal stayed on record: %+v", removed)
	}
}

// Subagent transcripts are bookkeeping, not conversations: a removed folder
// whose session dir holds only them has an empty archive, so it leaves no
// ghost row and its record retires at once.
func TestAnArchiveOfOnlySubagentTranscriptsIsNotAGhost(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	dir := testenv.TempDir(t)
	writeSessionAt(t, filepath.Join(SessionDirFor(dir), "subagent-run.jsonl"))
	if err := addRememberedWorkspace(t.Context(), dir); err != nil {
		t.Fatal(err)
	}

	hub := NewHub(HubOptions{})
	defer hub.Shutdown()
	srv := httptest.NewServer(operatorHandler(hub))
	defer srv.Close()

	resp := postRemoveWorkspace(t, srv, dir)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove = %d, want 204", resp.StatusCode)
	}
	for name, rows := range map[string][]treeWorkspace{"everyday": treeRoots(t, srv), "archived": treeRootsArchived(t, srv)} {
		if treeRowFor(rows, dir) != nil {
			t.Fatalf("bookkeeping alone kept a row for the removed folder in %s: %+v", name, rows)
		}
	}
	if removed := readListRemoved(t); slices.Contains(removed, projectIdentity(dir)) {
		t.Errorf("a removal with nothing left to show stayed on record: %+v", removed)
	}
}

// A tombstone written before removals carried the folder path — every record
// this build's predecessor left behind — still names a ghost row: the
// identity is the folder's canonical path, and it answers on disk as one.
// The conversation is filed under that spelling, since the legacy record is
// all the ghost has; a folder spelled differently falls back to silence.
func TestALegacyTombstoneWithoutAPathStillShowsItsArchive(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	dir := testenv.TempDir(t)
	writeSessionAt(t, filepath.Join(SessionDirFor(projectIdentity(dir)), "held.jsonl"))
	list, err := readWorkspaceList(workspacesPath())
	if err != nil {
		t.Fatal(err)
	}
	// Write the removal the old build made: the identity, and nothing else.
	list.Removed = []string{projectIdentity(dir)}
	list.RemovedPaths = nil
	if err := updateWorkspaceList(t.Context(), false, func(l *workspaceList) error {
		*l = list
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	hub := NewHub(HubOptions{})
	defer hub.Shutdown()
	srv := httptest.NewServer(operatorHandler(hub))
	defer srv.Close()

	ghost := treeRowFor(treeRootsArchived(t, srv), dir)
	if ghost == nil || !ghost.Removed {
		t.Fatalf("a legacy tombstone left no ghost row in the 已归档 scope: %+v", treeRootsArchived(t, srv))
	}
}

// Search reaches what the everyday list hides: an archived conversation of a
// removed project answers a matching query, under its ghost row, from the
// scope the sidebar searches with.
func TestSearchSurfacesARemovedProjectsArchive(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	dir := testenv.TempDir(t)
	path := filepath.Join(SessionDirFor(dir), "kept.jsonl")
	writeSessionAt(t, path)
	if err := addRememberedWorkspace(t.Context(), dir); err != nil {
		t.Fatal(err)
	}
	hub := NewHub(HubOptions{})
	defer hub.Shutdown()
	srv := httptest.NewServer(operatorHandler(hub))
	defer srv.Close()
	resp := postRemoveWorkspace(t, srv, dir)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove = %d, want 204", resp.StatusCode)
	}

	rows := hubGet[[]treeWorkspace](t, srv, "/tree?q=hello")
	ghost := treeRowFor(rows, dir)
	if ghost == nil || !ghost.Removed || len(ghost.Sessions) != 1 {
		t.Fatalf("a search over a removed project's archive found nothing: %+v", rows)
	}
}
