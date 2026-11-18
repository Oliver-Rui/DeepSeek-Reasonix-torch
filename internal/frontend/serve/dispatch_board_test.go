package serve

import (
	"testing"

	"reasonix/internal/session/control"
)

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

func TestDispatchBoardEmptyWhenNothingDispatched(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	f.sender(f.project("alpha", true))
	if rows := f.hub.dispatchBoard(); len(rows) != 0 {
		t.Fatalf("board rows = %+v, want empty", rows)
	}
}
