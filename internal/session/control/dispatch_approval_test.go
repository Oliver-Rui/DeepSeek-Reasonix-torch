package control

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/event"
)

// A session another session opened for a task has nobody at its window. A card
// there is not a question, it is a stall: the person is looking at the pane that
// dispatched it, and the dispatcher cannot answer either. So a command whose
// shape static analysis cannot read must run, not wait.
func TestADispatchedSessionRunsNestedShellWithoutACard(t *testing.T) {
	dir := testenv.TempDir(t)
	approvals := make(chan event.Approval, 4)
	c := New(Options{
		SessionDir: dir, SessionPath: filepath.Join(dir, "s.jsonl"),
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.ApprovalRequest {
				approvals <- e.Approval
			}
		}),
	})
	defer c.Close()
	c.SetToolApprovalMode(ToolApprovalAuto)
	c.MarkOpenedByDispatch("s-0000000000000009", []byte("secret-for-the-pair"), 1)

	gate := c.newInteractiveGate()
	for _, command := range []string{
		`powershell.exe -NoProfile -ExecutionPolicy Bypass -File "tools\gate\doclint.ps1"`,
		`bash -c "echo hello"`,
		`ls /d/ | head -30`,
	} {
		args, err := json.Marshal(map[string]string{"command": command})
		if err != nil {
			t.Fatal(err)
		}
		allow, reason, err := gate.Check(context.Background(), "bash", args, false)
		if err != nil || !allow || reason != "" {
			t.Errorf("dispatched command %q = (%v,%q,%v), want allow without a card", command, allow, reason, err)
		}
	}
	select {
	case approval := <-approvals:
		t.Fatalf("a dispatched session asked the person: %+v", approval)
	case <-time.After(50 * time.Millisecond):
	}
}

// The same session must not be given more than the dispatcher had. A rule that
// wants a person is refused with the reason, so the model learns why and goes on
// rather than waiting for an answer nobody will give.
func TestADispatchedSessionIsRefusedWhereAPersonIsRequired(t *testing.T) {
	dir := testenv.TempDir(t)
	c := New(Options{SessionDir: dir, SessionPath: filepath.Join(dir, "s.jsonl"), Sink: event.Discard})
	defer c.Close()
	c.SetToolApprovalMode(ToolApprovalAuto)
	c.MarkOpenedByDispatch("s-0000000000000009", []byte("secret-for-the-pair"), 1)

	gate := c.newInteractiveGate()
	args, err := json.Marshal(map[string]string{"text": "a fact worth keeping"})
	if err != nil {
		t.Fatal(err)
	}
	allow, reason, err := gate.Check(context.Background(), memoryRememberTool, args, false)
	if err != nil {
		t.Fatal(err)
	}
	if allow {
		t.Fatal("a dispatched session wrote memory with nobody to ask")
	}
	if reason == "" {
		t.Fatal("a refusal needs a reason the model can read and act on")
	}
}

// A session the person opened keeps the card: this is about the pane nobody is
// watching, not about making auto looser everywhere.
func TestAPersonOwnSessionStillAsks(t *testing.T) {
	dir := testenv.TempDir(t)
	approvals := make(chan event.Approval, 1)
	c := New(Options{
		SessionDir: dir, SessionPath: filepath.Join(dir, "s.jsonl"),
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.ApprovalRequest {
				select {
				case approvals <- e.Approval:
				default:
				}
			}
		}),
	})
	c.SetToolApprovalMode(ToolApprovalAuto)

	gate := c.newInteractiveGate()
	args, err := json.Marshal(map[string]string{"command": `bash -c "echo hello"`})
	if err != nil {
		t.Fatal(err)
	}
	// The check blocks on the card, so it runs on its own goroutine and is
	// released before the controller closes: a pending approval outliving the
	// session is a leak the race detector reports as this test's fault.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = gate.Check(context.Background(), "bash", args, false)
	}()
	select {
	case approval := <-approvals:
		c.Approve(approval.ID, true, false, false)
	case <-time.After(2 * time.Second):
		t.Fatal("a person's own session stopped asking about a nested shell")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the card was answered but the check never returned")
	}
	c.Close()
	c.autosaveWG.Wait()
}
