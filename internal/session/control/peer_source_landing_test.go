package control

import (
	"context"
	"path/filepath"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/provider"
	"reasonix/internal/runtime/agent"
)

// The source a host declares at delivery has to survive to the message the
// model's turn is about, because history renders the 来自「X」 label from it. A
// relayed session message runs as a synthetic continuation — the host opened
// the turn, the queue item only fills it — and that path rebuilds the boundary,
// so a rebuild that declares no source loses the label.
func TestASyntheticContinuationKeepsTheSourceItsHostDeclared(t *testing.T) {
	dir := testenv.TempDir(t)
	c := New(Options{SessionDir: dir, SessionPath: filepath.Join(dir, "s.jsonl")})
	defer c.Close()

	peer := &provider.PeerSource{SessionID: "s-other", Title: "组员B", Path: "/sessions/b.jsonl"}
	via := &provider.Via{Device: "phone"}
	ctx := c.withTurnTags(context.Background(), turnTags{via: via, peer: peer})

	boundary, ok := agent.HostTurnBoundaryFrom(c.continueAnnouncedTurn(ctx))
	if !ok {
		t.Fatal("the synthetic continuation declared no boundary at all")
	}
	if !boundary.HostAuthored {
		t.Fatal("a continuation is still the host's own message")
	}
	if boundary.Peer == nil || boundary.Peer.SessionID != peer.SessionID || boundary.Peer.Title != peer.Title || boundary.Peer.Path != peer.Path {
		t.Fatalf("the continuation dropped the source the host declared: %+v", boundary.Peer)
	}
	if boundary.Via == nil || boundary.Via.Device != via.Device {
		t.Fatalf("the continuation dropped the device the host declared: %+v", boundary.Via)
	}
}

// A turn no host declared a source for stays unattributed: the fix must not
// invent a source for a person's own typing.
func TestASyntheticContinuationInventsNoSource(t *testing.T) {
	dir := testenv.TempDir(t)
	c := New(Options{SessionDir: dir, SessionPath: filepath.Join(dir, "s.jsonl")})
	defer c.Close()

	boundary, ok := agent.HostTurnBoundaryFrom(c.continueAnnouncedTurn(context.Background()))
	if !ok {
		t.Fatal("the synthetic continuation declared no boundary at all")
	}
	if boundary.Peer != nil || boundary.Via != nil {
		t.Fatalf("a continuation with nothing declared carried %+v / %+v", boundary.Peer, boundary.Via)
	}
}
