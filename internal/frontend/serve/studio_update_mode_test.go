package serve

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"reasonix/internal/contract/config"
	"reasonix/internal/platform/update"
)

func updateMode(t *testing.T, base string) string {
	t.Helper()
	resp, err := http.Get(base + "/studio/update-mode")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /studio/update-mode = %d, %v", resp.StatusCode, err)
	}
	return body.Mode
}

func postUpdateMode(t *testing.T, base, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(base+"/studio/update-mode", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out.Code
}

func TestUpdateModeRoutesDefaultToNotifyAndWriteTheUsersFile(t *testing.T) {
	srv := studioServer(t, &update.Install{Version: "2.10.0"})
	if got := updateMode(t, srv.URL); got != "notify" {
		t.Fatalf("a fresh install reads %q, want notify", got)
	}
	if status, _ := postUpdateMode(t, srv.URL, `{"mode":"manual"}`); status != http.StatusOK {
		t.Fatalf("POST manual = %d", status)
	}
	if got := config.LoadForEdit(config.UserConfigPath()).DesktopUpdateMode(); got != "manual" {
		t.Fatalf("on disk %q, want manual", got)
	}
	if got := updateMode(t, srv.URL); got != "manual" {
		t.Fatalf("read back %q, want manual", got)
	}
	if status, _ := postUpdateMode(t, srv.URL, `{"mode":"download"}`); status != http.StatusOK || updateMode(t, srv.URL) != "download" {
		t.Fatal("download did not stick")
	}
}

func TestUpdateModeRouteRefusesAModeItDoesNotOfferByCode(t *testing.T) {
	srv := studioServer(t, &update.Install{Version: "2.10.0"})
	status, code := postUpdateMode(t, srv.URL, `{"mode":"auto"}`)
	if status != http.StatusBadRequest || code != codeUpdateModeInvalid {
		t.Fatalf("POST download = %d %q, want 400 %q", status, code, codeUpdateModeInvalid)
	}
	if got := updateMode(t, srv.URL); got != "notify" {
		t.Fatalf("a refused mode left %q behind", got)
	}
}

func TestUpdateModeRoutesAnswerWithoutAnInstall(t *testing.T) {
	// The fork's catalog reads GitHub without a declared install, and the
	// background watch still needs its mode: the routes answer either way.
	srv := studioServer(t, nil)
	resp, err := http.Get(srv.URL + "/studio/update-mode")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got.Mode == "" {
		t.Fatalf("GET answered no mode: %q", got.Mode)
	}
	resp, err = http.Post(srv.URL+"/studio/update-mode", "application/json", bytes.NewReader([]byte(`{"mode":"notify"}`)))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if saved.Mode != "notify" {
		t.Fatalf("POST answered mode %q", saved.Mode)
	}
	if got := updateMode(t, srv.URL); got != "notify" {
		t.Fatalf("the saved mode read back %q", got)
	}
}
