package control

import (
	"path/filepath"
	"testing"

	"reasonix/internal/base/testenv"
)

// The label a delivered message carries has to say which direction it came
// from: 来自「X」 for a task handed over, 「X」的回信 for an answer to one this
// session asked for. The only fact that tells them apart is who opened the
// pairing — this session (it handed the task out) or the sender (it was handed
// one) — so the label must read that, not the wording of the body.
func TestATaskThisSessionHandedOutIsAnsweredNotAnnounced(t *testing.T) {
	dir := testenv.TempDir(t)
	c := New(Options{SessionDir: dir, SessionPath: filepath.Join(dir, "s.jsonl")})
	defer c.Close()

	const child = "s-0000000000000002"
	if c.handedTaskTo(child) {
		t.Fatal("a session with no pairing is not one this one handed a task to")
	}
	c.NoteDispatchedSession(child, []byte("secret-for-the-pair"), "审查员A")
	if !c.handedTaskTo(child) {
		t.Fatal("a session this one opened for a task reads as handed out")
	}
}

// The other direction must not be mistaken for this one: a session that opened
// this one sent the first message, so its message is a handover, not an answer.
func TestASessionThatOpenedThisOneIsNotOneThisOneHandedTo(t *testing.T) {
	dir := testenv.TempDir(t)
	c := New(Options{SessionDir: dir, SessionPath: filepath.Join(dir, "s.jsonl")})
	defer c.Close()

	const parent = "s-0000000000000003"
	c.MarkOpenedByDispatch(parent, []byte("secret-for-the-pair"), 1)
	if c.handedTaskTo(parent) {
		t.Fatal("the session that opened this one reads as one it handed a task to")
	}
}
