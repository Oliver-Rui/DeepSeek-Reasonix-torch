package serve

import (
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
