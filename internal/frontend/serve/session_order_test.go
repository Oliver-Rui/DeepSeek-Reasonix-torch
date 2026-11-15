package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/state/sessionstore"
	"reasonix/internal/state/store"
)

func info(path string, at int, rank float64) sessionstore.SessionInfo {
	return sessionstore.SessionInfo{Path: path, LastActivityAt: time.Unix(int64(at), 0), ManualRank: rank}
}

func paths(order []sessionstore.SessionInfo) []string {
	out := make([]string, len(order))
	for i, s := range order {
		out[i] = s.Path
	}
	return out
}

func TestUnplacedSessionsComeFirstByActivityThenPlacedOnesByRank(t *testing.T) {
	list := []sessionstore.SessionInfo{info("old-placed-2", 1, 2048), info("new-unplaced", 9, 0), info("placed-1", 5, 1024), info("older-unplaced", 3, 0)}
	sortSessions(list)
	want := []string{"new-unplaced", "older-unplaced", "placed-1", "old-placed-2"}
	for i, p := range paths(list) {
		if p != want[i] {
			t.Fatalf("order = %v, want %v", paths(list), want)
		}
	}
}

func TestMovingASessionPlacesEveryUnplacedOneOnceAndKeepsTheVisibleOrder(t *testing.T) {
	list := []sessionstore.SessionInfo{info("a", 9, 0), info("b", 8, 0), info("c", 7, 0)}
	sortSessions(list)
	got, ok := rankUpdatesForStep(list, "c", -1)
	if !ok || len(got) != 3 {
		t.Fatalf("first placement ranks the whole list: %v %v", got, ok)
	}
	for i := range list {
		list[i].ManualRank = got[list[i].Path]
	}
	sortSessions(list)
	if w := []string{"a", "c", "b"}; paths(list)[0] != w[0] || paths(list)[1] != w[1] || paths(list)[2] != w[2] {
		t.Fatalf("after moving c up the order is %v, want %v", paths(list), w)
	}
}

func TestLaterMovesWriteOnlyTheMovedSessionWhileThereIsRoom(t *testing.T) {
	list := []sessionstore.SessionInfo{info("a", 1, 1024), info("b", 1, 2048), info("c", 1, 3072)}
	got, ok := rankUpdatesForStep(list, "c", -1)
	if !ok || len(got) != 1 || got["c"] != 1536 {
		t.Fatalf("moving c between a and b: %v %v", got, ok)
	}
	got, _ = rankUpdatesForStep(list, "a", 1)
	if len(got) != 1 || got["a"] != 2560 {
		t.Fatalf("moving a between b and c: %v", got)
	}
	got, _ = rankUpdatesForStep(list, "b", 1)
	if len(got) != 1 || got["b"] != 4096 {
		t.Fatalf("moving b to the end: %v", got)
	}
}

func TestRanksAreRenumberedWhenTheGapRunsOut(t *testing.T) {
	list := []sessionstore.SessionInfo{info("a", 1, 1024), info("b", 1, 1024.0005), info("c", 1, 4096)}
	got, ok := rankUpdatesForStep(list, "c", -1)
	if !ok || len(got) < 2 {
		t.Fatalf("a closed gap renumbers more than the moved one: %v %v", got, ok)
	}
	for i := range list {
		if rank, changed := got[list[i].Path]; changed {
			list[i].ManualRank = rank
		}
	}
	sortSessions(list)
	if p := paths(list); p[0] != "a" || p[1] != "c" || p[2] != "b" {
		t.Fatalf("after the renumbering the order is %v, want a c b", p)
	}
}

func TestAStepAtTheEdgeOrOnAnUnknownSessionChangesNothing(t *testing.T) {
	list := []sessionstore.SessionInfo{info("a", 1, 1024), info("b", 1, 2048)}
	if got, ok := rankUpdatesForStep(list, "a", -1); ok || got != nil {
		t.Fatalf("a is already first: %v %v", got, ok)
	}
	if got, ok := rankUpdatesForStep(list, "b", 1); ok || got != nil {
		t.Fatalf("b is already last: %v %v", got, ok)
	}
	if _, ok := rankUpdatesForStep(list, "nope", 1); ok {
		t.Fatal("an unknown session has no place to move")
	}
}

func TestOnlyConversationsInTheSameStateAreNeighbours(t *testing.T) {
	arch := info("archived", 5, 1024)
	arch.Archived = true
	list := []sessionstore.SessionInfo{info("a", 9, 2048), arch, info("b", 1, 3072)}
	sortSessions(list)
	got, ok := rankUpdatesForStep(list, "b", -1)
	if !ok || got["b"] >= 2048 {
		t.Fatalf("moving b up passes a, not the archived row between them: %v %v", got, ok)
	}
	if _, touched := got["archived"]; touched {
		t.Fatalf("a row that was not part of the move was rewritten: %v", got)
	}
}

func moveSessionRequest(t *testing.T, srv *httptest.Server, path string, direction int) *http.Response {
	t.Helper()
	resp, err := http.Post(srv.URL+"/tree/sessions/move", "application/json", strings.NewReader(fmt.Sprintf(`{"path":%q,"direction":%d}`, path, direction)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

func rowNames(ws treeWorkspace) []string {
	out := make([]string, len(ws.Sessions))
	for i, s := range ws.Sessions {
		out[i] = s.Name
	}
	return out
}

func TestMovingASessionChangesTheTreeAndSurvivesAReload(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	paths := seedAged(t, root, 4)

	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != "[session-003 session-002 session-001 session-000]" {
		t.Fatalf("by activity: %v", got)
	}
	if resp := moveSessionRequest(t, srv, paths[0], -1); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("move = %d", resp.StatusCode)
	}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != "[session-003 session-002 session-000 session-001]" {
		t.Fatalf("after moving the oldest up one: %v", got)
	}
	if resp := moveSessionRequest(t, srv, paths[3], 1); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("move = %d", resp.StatusCode)
	}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != "[session-002 session-003 session-000 session-001]" {
		t.Fatalf("after moving the newest down one: %v", got)
	}
	if resp := moveSessionRequest(t, srv, paths[2], -1); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("an edge step is not an error: %d", resp.StatusCode)
	}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != "[session-002 session-003 session-000 session-001]" {
		t.Fatalf("an edge step changed the order: %v", got)
	}
}

func TestAConversationCreatedAfterPlacingStartsOnTop(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	paths := seedAged(t, root, 3)
	moveSessionRequest(t, srv, paths[0], -1)
	fresh := filepath.Join(SessionDirFor(root), "session-new.jsonl")
	writeSessionAt(t, fresh)
	if got := rowNames(treeFor(t, srv, "")); got[0] != "session-new" {
		t.Fatalf("a conversation nobody placed belongs on top, got %v", got)
	}
}

func TestMovingRefusesAPathOutsideEveryWorkspaceAndABadDirection(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	seedAged(t, root, 2)
	if resp := moveSessionRequest(t, srv, filepath.Join(testenv.TempDir(t), "elsewhere.jsonl"), 1); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a path outside the workspaces = %d, want 403", resp.StatusCode)
	}
	inside := filepath.Join(SessionDirFor(root), "session-000.jsonl")
	if resp := moveSessionRequest(t, srv, inside, 2); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("direction 2 = %d, want 400", resp.StatusCode)
	}
}

// seedRunningSidecars leaves the disk the way a conversation mid-turn has it:
// the pane's own lease on the not-yet-written transcript is already held, and
// the inbox its steering writes to stands beside it. The listing never reads
// these; they are what the disk really looks like while a first turn is in
// flight.
func seedRunningSidecars(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path + ".lease.lock"); err != nil {
		if err := os.WriteFile(path+".lease.lock", nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path + ".lease.json"); err != nil {
		info := sessionstore.SessionLeaseInfo{
			SessionPath: path, WriterID: "test-writer", PID: os.Getpid(),
			Hostname: "localhost", AcquiredAt: time.Now().UTC(),
		}
		b, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".lease.json", b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(store.SessionInboxDir(path), 0o755); err != nil {
		t.Fatal(err)
	}
}

func rankOf(t *testing.T, path string) (float64, bool) {
	t.Helper()
	m, ok, err := sessionstore.LoadBranchMeta(path)
	if err != nil {
		t.Fatalf("read the sidecar of %s: %v", path, err)
	}
	return m.ManualRank, ok
}

// A conversation a pane is driving is the sidebar's first row before its
// transcript exists — only lease and inbox sidecars — and the move endpoint
// used to refuse exactly that row. The rows a move works on are the
// sidebar's: the in-flight one moves without a rank, a dispatch transcript
// takes part in nothing, and a listed step lands where the sidebar shows.
func TestMovingAConversationTheSidebarShows(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	rt := hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	paths := seedAged(t, root, 3)
	live := filepath.Join(SessionDirFor(root), "20261115-160810-in-progress.jsonl")
	rt.Server.Controller().SetSessionPath(live)
	seedRunningSidecars(t, live)
	dispatch := filepath.Join(SessionDirFor(root), "subagent-dispatch-202605171512.jsonl")
	writeSessionAt(t, dispatch)

	top := []string{"20261115-160810-in-progress", "session-002", "session-001", "session-000"}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != fmt.Sprint(top) {
		t.Fatalf("the sidebar rows are %v, want %v", got, top)
	}

	if resp := moveSessionRequest(t, srv, live, 1); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("moving the in-flight conversation = %d, want 204", resp.StatusCode)
	}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != fmt.Sprint(top) {
		t.Fatalf("after moving the in-flight conversation: %v, want %v", got, top)
	}
	if rank, _ := rankOf(t, live); rank != 0 {
		t.Fatalf("the in-flight conversation grew a rank %v with no transcript to place", rank)
	}

	// Moving the oldest listed conversation up one renumbers its group; the
	// dispatch transcript beside it stays out of the rows and out of the ranks.
	if resp := moveSessionRequest(t, srv, paths[0], -1); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("move = %d, want 204", resp.StatusCode)
	}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != fmt.Sprint([]string{"20261115-160810-in-progress", "session-002", "session-000", "session-001"}) {
		t.Fatalf("after moving the oldest up one the sidebar shows %v", got)
	}
	for path, want := range map[string]float64{paths[2]: 1024, paths[0]: 2048, paths[1]: 3072, dispatch: 0} {
		if rank, _ := rankOf(t, path); rank != want {
			t.Fatalf("%s carries rank %v, want %v", filepath.Base(path), rank, want)
		}
	}
	if resp := moveSessionRequest(t, srv, dispatch, 1); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a transcript no row shows = %d, want 403", resp.StatusCode)
	}

	// The in-flight conversation is listed too, so the sidebar's own acts on it
	// answer: pinning writes the sidecar, and a step upward is an edge, not an
	// error.
	pin := func(pinned bool) {
		t.Helper()
		body, err := json.Marshal(map[string]any{"path": live, "pinned": pinned})
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Post(srv.URL+"/tree/sessions/pin", "application/json", strings.NewReader(string(body)))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("pinning the in-flight conversation = %d, want 204", resp.StatusCode)
		}
	}
	pin(true)
	if m, ok, err := sessionstore.LoadBranchMeta(live); err != nil || !ok || !m.Pinned {
		t.Fatalf("the pin on the in-flight conversation did not land: ok=%v err=%v", ok, err)
	}
	pin(false)
	if resp := moveSessionRequest(t, srv, live, -1); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("an edge step on the in-flight conversation = %d, want 204", resp.StatusCode)
	}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != fmt.Sprint([]string{"20261115-160810-in-progress", "session-002", "session-000", "session-001"}) {
		t.Fatalf("the edge step disturbed the order: %v", got)
	}
}

// The archived conversations are a partition of their own in the rank math:
// moving the active ones writes nothing to an archived sidecar, and moving
// within the archived partition leaves the active ones alone.
func TestMovingStaysInsideTheArchivedPartition(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	paths := seedAged(t, root, 4)
	if err := sessionstore.SetSessionArchived(paths[0], true); err != nil {
		t.Fatal(err)
	}
	if err := sessionstore.SetSessionArchived(paths[3], true); err != nil {
		t.Fatal(err)
	}
	if got := rowNames(treeFor(t, srv, "")); fmt.Sprint(got) != fmt.Sprint([]string{"session-003", "session-002", "session-001", "session-000"}) {
		t.Fatalf("the sidebar rows are %v", got)
	}

	// The newest conversation is archived: its step down renumbers the
	// archived partition it belongs to and touches no active sidecar.
	if resp := moveSessionRequest(t, srv, paths[3], 1); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("move = %d, want 204", resp.StatusCode)
	}
	for path, want := range map[string]float64{paths[0]: 1024, paths[3]: 2048, paths[1]: 0, paths[2]: 0} {
		if rank, _ := rankOf(t, path); rank != want {
			t.Fatalf("%s carries rank %v, want %v", filepath.Base(path), rank, want)
		}
	}
	tree := treeFor(t, srv, "")
	// The comparator puts the never-placed first, so the two unplaced active
	// conversations lead, and the archived pair follows in the rank it was given.
	if got := rowNames(tree); fmt.Sprint(got) != fmt.Sprint([]string{"session-002", "session-001", "session-000", "session-003"}) {
		t.Fatalf("after the archived step the sidebar shows %v", got)
	}
	if tree.Sessions[0].Archived || tree.Sessions[1].Archived {
		t.Fatalf("an active conversation read back as archived: %+v %+v", tree.Sessions[0], tree.Sessions[1])
	}
	if !tree.Sessions[2].Archived || !tree.Sessions[3].Archived {
		t.Fatalf("an archived conversation lost its state: %+v %+v", tree.Sessions[2], tree.Sessions[3])
	}
}
