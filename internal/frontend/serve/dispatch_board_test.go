package serve

import (
	"testing"

	"reasonix/internal/session/control"
	"reasonix/internal/state/sessionstore"
)

func archiveSessionForTest(t *testing.T, path string) {
	t.Helper()
	m, _, err := sessionstore.LoadBranchMeta(path)
	if err != nil {
		t.Fatal(err)
	}
	m.Archived = true
	if err := sessionstore.SaveBranchMeta(path, m); err != nil {
		t.Fatal(err)
	}
}

func TestDispatchBoardCollectsBothEndsOfADispatch(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	billing := f.project("billing", true)
	got, err := sender.DispatchToProject(t.Context(), billing, "rotate the key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	pane := f.paneOf(got.Key)
	writeSessionAt(t, pane.SessionPath())

	rows := f.hub.dispatchBoard()
	if len(rows) != 1 {
		t.Fatalf("board rows = %+v, want one", rows)
	}
	row := rows[0]
	if row.Assignee.SessionPath != pane.SessionPath() || row.Assignee.Project != billing {
		t.Fatalf("assignee = %+v, want path %q in %q", row.Assignee, pane.SessionPath(), billing)
	}
	if row.Dispatcher.SessionPath != sender.SessionPath() {
		t.Fatalf("dispatcher = %+v, want %q", row.Dispatcher, sender.SessionPath())
	}
	if row.Role != "reviewer" || row.State != string(control.PeerWaiting) {
		t.Fatalf("row = %+v, want role reviewer and waiting", row)
	}
	if !row.CrossProject {
		t.Fatalf("row = %+v, want crossProject across alpha and billing", row)
	}
}

func TestDedupeBoardRepliesKeepsTheRoleBearingRow(t *testing.T) {
	rows := []DispatchBoardRow{
		{Dispatcher: DispatchBoardParty{SessionPath: "/member"}, Assignee: DispatchBoardParty{SessionPath: "/lead"}, State: "working"},
		{Dispatcher: DispatchBoardParty{SessionPath: "/lead"}, Assignee: DispatchBoardParty{SessionPath: "/member"}, Role: "研究员", State: "replied"},
	}
	kept := dedupeBoardReplies(rows)
	if len(kept) != 1 || kept[0].Role != "研究员" || kept[0].Dispatcher.SessionPath != "/lead" {
		t.Fatalf("kept = %+v, want the single role-bearing dispatch", kept)
	}
}

func TestDedupeBoardRepliesKeepsUnpairedMessages(t *testing.T) {
	rows := []DispatchBoardRow{
		{Dispatcher: DispatchBoardParty{SessionPath: "/a"}, Assignee: DispatchBoardParty{SessionPath: "/b"}, State: "waiting"},
	}
	if kept := dedupeBoardReplies(rows); len(kept) != 1 {
		t.Fatalf("kept = %+v, want the row kept when no role-bearing reverse exists", kept)
	}
}

func TestDispatchBoardHidesArchivedEnds(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	billing := f.project("billing", true)
	got, err := sender.DispatchToProject(t.Context(), billing, "rotate the key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	pane := f.paneOf(got.Key)
	writeSessionAt(t, pane.SessionPath())
	if rows := f.hub.dispatchBoard(); len(rows) != 1 {
		t.Fatalf("board before archive = %+v, want one", rows)
	}

	archiveSessionForTest(t, pane.SessionPath())
	if rows := f.hub.dispatchBoard(); len(rows) != 0 {
		t.Fatalf("board after archiving the assignee = %+v, want empty", rows)
	}
}

func TestDispatchBoardEmptyWhenNothingDispatched(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	f.sender(f.project("alpha", true))
	if rows := f.hub.dispatchBoard(); len(rows) != 0 {
		t.Fatalf("board rows = %+v, want empty", rows)
	}
	if builds := f.hub.board.buildCount(); builds != 0 {
		t.Fatalf("catalog builds = %d, want none for an idle machine", builds)
	}
}

func TestDispatchBoardReusesOneCatalogBuild(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	billing := f.project("billing", true)
	got, err := sender.DispatchToProject(t.Context(), billing, "rotate the key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	writeSessionAt(t, f.paneOf(got.Key).SessionPath())

	if rows := f.hub.dispatchBoard(); len(rows) != 1 {
		t.Fatalf("first board = %+v, want one", rows)
	}
	if rows := f.hub.dispatchBoard(); len(rows) != 1 {
		t.Fatalf("second board = %+v, want one", rows)
	}
	if builds := f.hub.board.buildCount(); builds != 1 {
		t.Fatalf("catalog builds = %d, want one walk shared by both reads", builds)
	}
}

func TestDispatchBoardArchivedEndDoesNotRebuildCatalog(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	billing := f.project("billing", true)
	got, err := sender.DispatchToProject(t.Context(), billing, "rotate the key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	pane := f.paneOf(got.Key)
	writeSessionAt(t, pane.SessionPath())
	if rows := f.hub.dispatchBoard(); len(rows) != 1 {
		t.Fatalf("board before archive = %+v, want one", rows)
	}

	archiveSessionForTest(t, pane.SessionPath())
	if rows := f.hub.dispatchBoard(); len(rows) != 0 {
		t.Fatalf("board after archive = %+v, want empty", rows)
	}
	if builds := f.hub.board.buildCount(); builds != 1 {
		t.Fatalf("catalog builds = %d, want the sidecar recheck without a new walk", builds)
	}
}

func TestDispatchBoardRescansForANewTarget(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	first := f.project("billing", true)
	got, err := sender.DispatchToProject(t.Context(), first, "rotate the key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	writeSessionAt(t, f.paneOf(got.Key).SessionPath())
	if rows := f.hub.dispatchBoard(); len(rows) != 1 {
		t.Fatalf("first board = %+v, want one", rows)
	}
	if builds := f.hub.board.buildCount(); builds != 1 {
		t.Fatalf("catalog builds = %d, want one", builds)
	}

	second := f.project("gamma", true)
	next, err := sender.DispatchToProject(t.Context(), second, "second task", "reviewer-2")
	if err != nil {
		t.Fatal(err)
	}
	writeSessionAt(t, f.paneOf(next.Key).SessionPath())
	if rows := f.hub.dispatchBoard(); len(rows) != 2 {
		t.Fatalf("board after a new target = %+v, want two", rows)
	}
	if builds := f.hub.board.buildCount(); builds != 2 {
		t.Fatalf("catalog builds = %d, want a forced walk for the new target", builds)
	}
}
