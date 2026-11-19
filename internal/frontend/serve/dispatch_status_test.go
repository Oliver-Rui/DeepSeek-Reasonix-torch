package serve

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/config"
	"reasonix/internal/session/control"
	"reasonix/internal/state/peermail"
	"reasonix/internal/state/sessionstore"
)

// seedThreadAudit writes the audit trail a sender would leave for one dispatch,
// from the sender's own peer key to a target.
func seedThreadAudit(t *testing.T, peersDir, senderPath, targetPath string, records func(sender, target string) []peermail.AuditRecord) {
	t.Helper()
	dir, err := peermail.Open(peersDir)
	if err != nil {
		t.Fatal(err)
	}
	sender := control.PeerKeyForPath(senderPath)
	target := control.PeerKeyForPath(targetPath)
	when := time.Now().Add(-time.Minute)
	for _, r := range records(sender, target) {
		if r.At.IsZero() {
			r.At = when
			when = when.Add(time.Second)
		}
		if err := dir.Audit(r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDispatchThreadsStatusFoldsEachState(t *testing.T) {
	for name, tc := range map[string]struct {
		records func(sender, target string) []peermail.AuditRecord
		want    control.PeerThreadState
	}{
		"waiting": {func(s, tr string) []peermail.AuditRecord {
			return []peermail.AuditRecord{{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent}}
		}, control.PeerWaiting},
		"working": {func(s, tr string) []peermail.AuditRecord {
			return []peermail.AuditRecord{
				{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent},
				{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionReleased},
			}
		}, control.PeerWorking},
		"declined": {func(s, tr string) []peermail.AuditRecord {
			return []peermail.AuditRecord{
				{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent},
				{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionDenied},
			}
		}, control.PeerDeclined},
		"replied": {func(s, tr string) []peermail.AuditRecord {
			return []peermail.AuditRecord{
				{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent},
				{Receiver: s, Sender: tr, ID: "r1", Decision: peermail.DecisionHeld},
			}
		}, control.PeerReplied},
	} {
		t.Run(name, func(t *testing.T) {
			home := newDispatchHome(t)
			senderPath := sessionstore.NewSessionPath(filepath.Join(home, "sessions"), "sender")
			targetPath := sessionstore.NewSessionPath(filepath.Join(home, "sessions"), "target")
			h := NewHub(HubOptions{})
			seedThreadAudit(t, filepath.Join(home, "peers"), senderPath, targetPath, tc.records)
			rows := h.dispatchThreadsStatus(senderPath)
			if len(rows) != 1 || rows[0].State != tc.want || rows[0].Since.IsZero() {
				t.Fatalf("rows = %+v, want one %s row with a since", rows, tc.want)
			}
		})
	}
}

func TestDispatchThreadsStatusRebuildsAfterRestart(t *testing.T) {
	home := newDispatchHome(t)
	senderPath := sessionstore.NewSessionPath(filepath.Join(home, "sessions"), "sender")
	targetPath := sessionstore.NewSessionPath(filepath.Join(home, "sessions"), "target")
	seedThreadAudit(t, filepath.Join(home, "peers"), senderPath, targetPath, func(s, tr string) []peermail.AuditRecord {
		return []peermail.AuditRecord{
			{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent},
			{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionApproved},
		}
	})
	// A fresh Hub over the same state root, with no in-memory book, reads the
	// state back from the audit trail.
	h := NewHub(HubOptions{})
	rows := h.dispatchThreadsStatus(senderPath)
	if len(rows) != 1 || rows[0].State != control.PeerWorking {
		t.Fatalf("after restart rows = %+v, want one working row from the audit", rows)
	}
}

func TestDispatchThreadsStatusFallsBackToBookWithoutAudit(t *testing.T) {
	home := newDispatchHome(t)
	h := NewHub(HubOptions{})
	senderPath := sessionstore.NewSessionPath(filepath.Join(home, "sessions"), "sender")
	targetPath := sessionstore.NewSessionPath(filepath.Join(home, "sessions"), "target")
	// The book names a session that exists: a dispatch whose target file never
	// landed is the case the delete check hides, and this test is about the
	// fallback that runs before any of that.
	writeSessionFile(t, targetPath)
	h.dispatch.noteThread(senderPath, "reviewer", targetPath, control.PeerKeyForPath(targetPath))
	rows := h.dispatchThreadsStatus(senderPath)
	if len(rows) != 1 || rows[0].Role != "reviewer" || rows[0].SessionPath != targetPath ||
		rows[0].State != control.PeerWaiting || rows[0].Since.IsZero() {
		t.Fatalf("fallback row = %+v, want book role/path with waiting state", rows)
	}
}

func TestDispatchThreadsStatusJoinsTargetTitleAndProject(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	sender := f.sender(f.project("alpha", true))
	b := f.project("billing", true)
	got, err := sender.DispatchToProject(t.Context(), b, "rotate the key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	pane := f.paneOf(got.Key)
	writeSessionAt(t, pane.SessionPath())
	rows := f.hub.dispatchThreadsStatus(sender.SessionPath())
	if len(rows) != 1 {
		t.Fatalf("rows = %+v, want one", rows)
	}
	if rows[0].Role != "reviewer" || rows[0].TargetProject != b {
		t.Fatalf("row = %+v, want role reviewer and project %q", rows[0], b)
	}
	if rows[0].TargetTitle == "" {
		t.Fatalf("row = %+v, want a joined target title", rows[0])
	}
}

// newDispatchHome sets a fresh state root so peersDir resolves under it.
func newDispatchHome(t *testing.T) string {
	t.Helper()
	home := testenv.TempDir(t)
	t.Setenv("REASONIX_HOME", home)
	if cfg, err := config.Load(); err == nil {
		if got := cfg.Roots().PeersDir(); !filepath.IsAbs(got) {
			t.Fatalf("PeersDir = %q, want absolute", got)
		}
	}
	return home
}

// writeSessionFile puts a minimal transcript where a session would be, so the
// catalog lists it and the archive check has a file to read.
func writeSessionFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"role":"user","content":"hi"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A target the person archived is a closed chapter: the audit trail still
// records the message that opened it, so the card would otherwise keep a link
// into a session the list no longer shows.
func TestDispatchThreadsDropsAnArchivedTarget(t *testing.T) {
	home := newDispatchHome(t)
	dir := config.ProjectSessionDir(filepath.Join(home, "project"))
	senderPath := sessionstore.NewSessionPath(dir, "sender")
	targetPath := sessionstore.NewSessionPath(dir, "target")
	writeSessionFile(t, senderPath)
	writeSessionFile(t, targetPath)
	if err := sessionstore.SaveBranchMeta(targetPath, sessionstore.BranchMeta{CustomTitle: "审查员A", Archived: true}); err != nil {
		t.Fatal(err)
	}
	h := NewHub(HubOptions{})
	seedThreadAudit(t, filepath.Join(home, "peers"), senderPath, targetPath, func(s, tr string) []peermail.AuditRecord {
		return []peermail.AuditRecord{{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent}}
	})
	if rows := h.dispatchThreadsStatus(senderPath); len(rows) != 0 {
		t.Fatalf("rows = %+v, want none for an archived target", rows)
	}
}

// The same for a target that is gone: a deleted session must not keep a link.
func TestDispatchThreadsDropsADeletedTarget(t *testing.T) {
	home := newDispatchHome(t)
	dir := config.ProjectSessionDir(filepath.Join(home, "project"))
	senderPath := sessionstore.NewSessionPath(dir, "sender")
	targetPath := sessionstore.NewSessionPath(dir, "target")
	writeSessionFile(t, senderPath)
	writeSessionFile(t, targetPath)
	h := NewHub(HubOptions{})
	seedThreadAudit(t, filepath.Join(home, "peers"), senderPath, targetPath, func(s, tr string) []peermail.AuditRecord {
		return []peermail.AuditRecord{{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent}}
	})
	// The catalog is built before the removal, so the join still knows the row.
	h.dispatchCatalog(true)
	if err := os.Remove(targetPath); err != nil {
		t.Fatal(err)
	}
	if rows := h.dispatchThreadsStatus(senderPath); len(rows) != 0 {
		t.Fatalf("rows = %+v, want none for a deleted target", rows)
	}
}

// A target that is still there keeps its link, so the filter cannot pass by
// dropping every row.
func TestDispatchThreadsKeepsALiveTarget(t *testing.T) {
	home := newDispatchHome(t)
	dir := config.ProjectSessionDir(filepath.Join(home, "project"))
	senderPath := sessionstore.NewSessionPath(dir, "sender")
	targetPath := sessionstore.NewSessionPath(dir, "target")
	writeSessionFile(t, senderPath)
	writeSessionFile(t, targetPath)
	h := NewHub(HubOptions{})
	seedThreadAudit(t, filepath.Join(home, "peers"), senderPath, targetPath, func(s, tr string) []peermail.AuditRecord {
		return []peermail.AuditRecord{{Receiver: tr, Sender: s, ID: "m1", Decision: peermail.DecisionSent}}
	})
	if rows := h.dispatchThreadsStatus(senderPath); len(rows) != 1 {
		t.Fatalf("rows = %+v, want the live target's link", rows)
	}
}
