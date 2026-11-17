package serve

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/state/sessionstore"
)

func postDispatch(t *testing.T, srv *httptest.Server, body any) []dispatchTargetResult {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(srv.URL+"/tree/sessions/dispatch", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /tree/sessions/dispatch = %d, want 200", res.StatusCode)
	}
	var out []dispatchTargetResult
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDispatchEndpointNewTarget(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	f.project("billing", true)
	sender := f.sender(f.project("alpha", true))
	srv := httptest.NewServer(operatorHandler(f.hub))
	defer srv.Close()

	results := postDispatch(t, srv, map[string]any{
		"from": sender.SessionPath(),
		"targets": []map[string]any{
			{"mode": "new", "project": "billing", "role": "reviewer", "instructions": "Focus on tests", "text": "rotate the key"},
		},
	})
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("results = %+v, want one ok", results)
	}
	thread := f.hub.dispatch.threadsFrom(sender.SessionPath())
	if len(thread) == 0 || thread[0].Role != "reviewer" {
		t.Fatalf("thread = %+v, want the reviewer role recorded", thread)
	}
}

func TestDispatchEndpointExistingTargetSpoolsAndPairs(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	alpha := f.project("alpha", true)
	sender := f.sender(alpha)
	target := f.sender(f.project("billing", true))
	writeSessionAt(t, target.SessionPath())
	srv := httptest.NewServer(operatorHandler(f.hub))
	defer srv.Close()

	results := postDispatch(t, srv, map[string]any{
		"from": sender.SessionPath(),
		"targets": []map[string]any{
			{"mode": "existing", "sessionPath": target.SessionPath(), "role": "auditor", "text": "review the logs"},
		},
	})
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("results = %+v, want one ok", results)
	}
	target.RefreshPeers()
	// The target was paired as a dispatch partner, so the task is taken by
	// dispatch consent without a card rather than left held.
	if held := target.HeldPeerMessages(); len(held) != 0 {
		t.Fatalf("held = %+v, want the paired task taken without a card", held)
	}
	if role := sessionstore.LoadBranchRoleText(target.SessionPath()); role != "auditor" {
		t.Fatalf("target role = %q, want auditor", role)
	}
	thread := f.hub.dispatch.threadsFrom(sender.SessionPath())
	if len(thread) == 0 || thread[0].SessionPath != target.SessionPath() {
		t.Fatalf("thread = %+v, want the target recorded", thread)
	}
	// The pairing is mutual: the target's reply returns to the sender without a
	// card too.
	if _, err := target.SendPeerMessage(sender.PeerKey(), "done"); err != nil {
		t.Fatal(err)
	}
	sender.RefreshPeers()
	if held := sender.HeldPeerMessages(); len(held) != 0 {
		t.Fatalf("sender held = %+v, want the reply taken by dispatch consent", held)
	}
}

func TestDispatchEndpointPerTargetRefusals(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	alpha := f.project("alpha", true)
	sender := f.sender(alpha)
	self := sender.SessionPath()
	writeSessionAt(t, self)
	srv := httptest.NewServer(operatorHandler(f.hub))
	defer srv.Close()

	results := postDispatch(t, srv, map[string]any{
		"from": self,
		"targets": []map[string]any{
			{"mode": "existing", "sessionPath": filepath.Join(alpha, "sessions", "does-not-exist.jsonl"), "role": "x", "text": "t"},
			{"mode": "existing", "sessionPath": self, "role": "x", "text": "t"},
			{"mode": "bogus", "text": "t"},
		},
	})
	if len(results) != 3 {
		t.Fatalf("results = %+v, want three", results)
	}
	if results[0].OK || results[0].Refusal == nil || results[0].Refusal.Code != codeDispatchTargetUnknown {
		t.Fatalf("unknown target = %+v", results[0])
	}
	if results[1].OK || results[1].Refusal == nil || results[1].Refusal.Code != codeDispatchSelf {
		t.Fatalf("self target = %+v", results[1])
	}
	if results[2].OK || results[2].Refusal == nil || results[2].Refusal.Code != codeBadValue {
		t.Fatalf("bad mode = %+v", results[2])
	}
}

func TestDispatchEndpointRejectsFromNotOpen(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	srv := httptest.NewServer(operatorHandler(f.hub))
	defer srv.Close()
	raw, _ := json.Marshal(map[string]any{
		"from":    filepath.Join(testenv.TempDir(t), "ghost.jsonl"),
		"targets": []map[string]any{{"mode": "new", "project": "x", "text": "t"}},
	})
	res, err := http.Post(srv.URL+"/tree/sessions/dispatch", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
}

func TestDispatchEndpointDuplicatesRefused(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	f.project("billing", true)
	sender := f.sender(f.project("alpha", true))
	srv := httptest.NewServer(operatorHandler(f.hub))
	defer srv.Close()
	target := map[string]any{"mode": "new", "project": "billing", "role": "reviewer", "text": "t"}
	results := postDispatch(t, srv, map[string]any{
		"from":    sender.SessionPath(),
		"targets": []map[string]any{target, target},
	})
	if len(results) != 2 || !results[0].OK || results[1].OK ||
		results[1].Refusal == nil || results[1].Refusal.Code != codeDispatchDuplicate {
		t.Fatalf("results = %+v, want first ok and duplicate refused", results)
	}
}
