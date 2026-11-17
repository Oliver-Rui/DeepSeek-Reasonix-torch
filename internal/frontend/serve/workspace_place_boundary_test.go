package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"reasonix/internal/base/testenv"
)

func placeWorkspaceRequest(t *testing.T, srv *httptest.Server, path, before string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"path": path, "before": before})
	if err != nil {
		t.Fatal(err)
	}
	return postLaunchJSON(t, srv.URL+"/tree/workspaces/move", string(body), nil)
}

func TestWorkspacePlaceKeepsThePinnedBoundary(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	a, b, c := testenv.TempDir(t), testenv.TempDir(t), testenv.TempDir(t)
	rememberWorkspace(a)
	rememberWorkspace(b)
	rememberWorkspace(c)
	if err := updateWorkspaceList(context.Background(), true, func(list *workspaceList) error {
		list.Pinned = []string{a, b}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got, want := Workspaces(), []string{b, a, c}; !slices.Equal(got, want) {
		t.Fatalf("pinned display = %v, want %v", got, want)
	}
	h := NewHub(HubOptions{})
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	if status, code := placeWorkspaceRequest(t, srv, a, b); status != http.StatusNoContent {
		t.Fatalf("within pinned = %d %s", status, code)
	}
	if got, want := Workspaces(), []string{a, b, c}; !slices.Equal(got, want) {
		t.Fatalf("after moving a above b = %v, want %v", got, want)
	}
	if status, code := placeWorkspaceRequest(t, srv, b, c); status != http.StatusConflict || code != codeWorkspaceOrderBoundary {
		t.Fatalf("crossing the pin boundary = %d %s, want 409 %s", status, code, codeWorkspaceOrderBoundary)
	}
	if status, code := placeWorkspaceRequest(t, srv, a, ""); status != http.StatusNoContent {
		t.Fatalf("pinned end = %d %s", status, code)
	}
	if got, want := Workspaces(), []string{b, a, c}; !slices.Equal(got, want) {
		t.Fatalf("after moving a to the pinned end = %v, want %v", got, want)
	}
}

func TestPinnedWorkspaceIdentityFollowsTheGroup(t *testing.T) {
	root, copy := projectWithWorktree(t)
	rememberWorkspace(root)
	rememberWorkspace(copy)
	if err := updateWorkspaceList(context.Background(), true, func(list *workspaceList) error {
		list.Pinned = []string{copy}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	h := NewHub(HubOptions{})
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()
	rows := hubGet[[]treeWorkspace](t, srv, "/tree")
	if len(rows) != 1 || !rows[0].Pinned {
		t.Fatalf("a pin on any member spelling must pin the grouped row: %+v", rows)
	}
}
