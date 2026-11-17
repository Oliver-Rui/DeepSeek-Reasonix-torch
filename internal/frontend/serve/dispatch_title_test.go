package serve

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGuidedTitlePromptCarriesRoleAndTask(t *testing.T) {
	h, rt, _, _ := titleSession(t)
	prov := &recordingTitleProvider{}
	rt.Server.titleModel.prov = prov
	title, err := rt.Server.tryGenerateGuidedTitle(t.Context(), "rotate the signing key", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if title == "" {
		t.Fatal("guided title was empty")
	}
	if prov.count() != 1 {
		t.Fatalf("provider requests = %d, want 1", prov.count())
	}
	req := prov.at(0)
	system := req.Messages[0].Content
	if !strings.Contains(system, "reviewer") {
		t.Fatalf("system prompt missing the role guidance:\n%s", system)
	}
	user := req.Messages[len(req.Messages)-1].Content
	if !strings.Contains(user, "rotate the signing key") {
		t.Fatalf("user prompt missing the task text:\n%s", user)
	}
	_ = h
}

func TestGuidedTitleFallsBackToRoleWhenNoProvider(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	root := f.project("alpha", true)
	sender := f.sender(root)
	target := f.sender(f.project("billing", true))
	targetPath := target.SessionPath()
	writeSessionAt(t, targetPath)
	// No title model is configured in the fixture, so guidance naming is
	// unavailable and the role must become the explicit title.
	f.hub.autoNameDispatchTarget(t.Context(), targetPath, dispatchTarget{Role: "reviewer", Text: "rotate the key"})
	cache := f.hub.titleCacheFor(filepath.Dir(targetPath))
	if got := cache.display(filepath.Base(targetPath), "", 0, ""); got != "reviewer" {
		t.Fatalf("fallback title = %q, want reviewer", got)
	}
	_ = sender
}

func TestGuidedTitlePromptUnchangedWithoutGuidance(t *testing.T) {
	if guidedTitlePrompt("") != titlePrompt {
		t.Fatal("an empty guidance must leave the ordinary title prompt byte-identical")
	}
	if !strings.Contains(guidedTitlePrompt("reviewer"), "reviewer") {
		t.Fatal("non-empty guidance must enter the prompt")
	}
}
