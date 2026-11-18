package serve

import (
	"testing"

	"reasonix/internal/session/control"
)

// runtimeForPath finds a fixture's runtime by its controller's session path.
func runtimeForPath(t *testing.T, h *Hub, path string) *Runtime {
	t.Helper()
	for _, rt := range h.Runtimes() {
		if c, ok := rt.Server.Controller().(*control.Controller); ok && c.SessionPath() == path {
			return rt
		}
	}
	t.Fatalf("no runtime for session %q", path)
	return nil
}

func TestRenameSessionForOwnConversation(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	writeSessionAt(t, sender.SessionPath())
	rt := runtimeForPath(t, f.hub, sender.SessionPath())
	got, err := f.hub.renameSessionFor(rt, "", "组长的新名字")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "组长的新名字" || got.Target != "self" || got.Path != sender.SessionPath() {
		t.Fatalf("receipt = %+v", got)
	}
}

func TestRenameSessionForDispatchedMember(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	billing := f.project("billing", true)
	got, err := sender.DispatchToProject(t.Context(), billing, "rotate the key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	pane := f.paneOf(got.Key)
	writeSessionAt(t, pane.SessionPath())
	rt := runtimeForPath(t, f.hub, sender.SessionPath())

	renamed, err := f.hub.renameSessionFor(rt, got.Key, "评审员")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Title != "评审员" || renamed.Target != "member" || renamed.Path != pane.SessionPath() {
		t.Fatalf("receipt = %+v", renamed)
	}
}

func TestRenameSessionForAForeignKeyIsRefused(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	rt := runtimeForPath(t, f.hub, sender.SessionPath())
	if _, err := f.hub.renameSessionFor(rt, "s-unknown-member", "随便"); err == nil {
		t.Fatal("renaming a session never dispatched to must be refused")
	}
}

func TestRenameSessionRefusesAnEmptyName(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	rt := runtimeForPath(t, f.hub, sender.SessionPath())
	if _, err := f.hub.renameSessionFor(rt, "", "   "); err == nil {
		t.Fatal("an empty name must be refused")
	}
}
