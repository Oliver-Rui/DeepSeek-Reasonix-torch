package serve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reasonix/internal/platform/update"
)

// The fork's version catalog is the GitHub releases list itself. A fake answers
// the API's shape; the mapping is what the test holds: newest first, the running
// build marked, the installer asset carried beside its row, and a dead GitHub
// answering with err beside the data rather than instead of it.
func studioFake(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/releases") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func withStudioRepo(t *testing.T, srv *httptest.Server) {
	t.Helper()
	wasRepo, wasBase := studioRepo, studioAPIBase
	studioRepo = "Oliver-Rui/DeepSeek-Reasonix-torch"
	studioAPIBase = srv.URL
	t.Cleanup(func() { studioRepo, studioAPIBase = wasRepo, wasBase })
}

func TestStudioCatalogMapsTheReleasesList(t *testing.T) {
	srv := studioFake(t, `[
		{"tag_name":"studio-v2.64.1","published_at":"2026-11-16T00:00:00Z","body":"fixes",
		 "assets":[{"name":"ReasonixStudio-2.64.1-windows-amd64-installer.exe","browser_download_url":"https://example.test/2.64.1.exe"}]},
		{"tag_name":"studio-v2.64.0","published_at":"2026-11-15T00:00:00Z","body":"",
		 "assets":[{"name":"ReasonixStudio-2.64.0-windows-amd64-installer.exe","browser_download_url":"https://example.test/2.64.0.exe"}]},
		{"tag_name":"studio-v2.63.1","published_at":"2026-11-10T00:00:00Z","body":"older"},
		{"tag_name":"not-a-release","published_at":"2026-11-09T00:00:00Z"},
		{"tag_name":"studio-v2.99.0","draft":true,"published_at":"2026-11-08T00:00:00Z"}]`)
	withStudioRepo(t, srv)
	t.Cleanup(resetStudioCatalogCache)

	hub := studioCatalog(context.Background(), "2.64.0", true)
	if hub.Err != "" {
		t.Fatalf("err = %q", hub.Err)
	}
	if hub.Current != "2.64.0" || hub.Latest != "2.64.1" || !hub.Newer {
		t.Fatalf("hub = %+v", hub)
	}
	if len(hub.Versions) != 3 {
		t.Fatalf("versions = %+v", hub.Versions)
	}
	first := hub.Versions[0]
	if first.Version != "2.64.1" || first.URL != "https://example.test/2.64.1.exe" || !first.HasNotes {
		t.Fatalf("first row = %+v", first)
	}
	if hub.Versions[1].Version != "2.64.0" || !hub.Versions[1].Current || hub.Versions[1].URL != "https://example.test/2.64.0.exe" {
		t.Fatalf("current row = %+v", hub.Versions[1])
	}
	if hub.Versions[2].Version != "2.63.1" || !hub.Versions[2].Older || !hub.Versions[2].HasNotes {
		t.Fatalf("older row = %+v", hub.Versions[2])
	}
}

func TestStudioCatalogOrdersNumericallyNotLexically(t *testing.T) {
	srv := studioFake(t, `[
		{"tag_name":"studio-v2.9.0","published_at":"2026-11-01T00:00:00Z"},
		{"tag_name":"studio-v2.10.0","published_at":"2026-11-02T00:00:00Z"}]`)
	withStudioRepo(t, srv)
	t.Cleanup(resetStudioCatalogCache)

	hub := studioCatalog(context.Background(), "2.9.0", true)
	if hub.Latest != "2.10.0" || !hub.Newer {
		t.Fatalf("hub = %+v", hub)
	}
}

func TestStudioCatalogCarriesErrBesideTheCurrentVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	withStudioRepo(t, srv)
	t.Cleanup(resetStudioCatalogCache)

	hub := studioCatalog(context.Background(), "2.64.0", true)
	if hub.Err == "" || hub.Current != "2.64.0" || len(hub.Versions) != 0 {
		t.Fatalf("hub = %+v", hub)
	}
}

func TestStudioReleaseNotesReadsTheBody(t *testing.T) {
	srv := studioFake(t, `[
		{"tag_name":"studio-v2.64.1","published_at":"2026-11-16T00:00:00Z","body":"## 修复"},
		{"tag_name":"studio-v2.64.0","published_at":"2026-11-15T00:00:00Z","body":"  "}]`)
	withStudioRepo(t, srv)
	t.Cleanup(resetStudioCatalogCache)

	if body, ok := studioReleaseNotes(context.Background(), "2.64.1", true); !ok || body != "## 修复" {
		t.Fatalf("notes = %q, %v", body, ok)
	}
	if _, ok := studioReleaseNotes(context.Background(), "2.64.0", false); ok {
		t.Fatalf("an empty body must not read as notes")
	}
}

func resetStudioCatalogCache() {
	studioCatalogCache.mu.Lock()
	defer studioCatalogCache.mu.Unlock()
	studioCatalogCache.releases = nil
	studioCatalogCache.at = time.Time{}
}

func TestStudioInstallerURLPrefersTheVersionedName(t *testing.T) {
	rel := studioRelease{Assets: []studioAsset{
		{Name: "ReasonixStudio-2.64.1-windows-amd64-installer.exe", BrowserDownloadURL: "https://example.test/a"},
		{Name: "other.exe", BrowserDownloadURL: "https://example.test/b"},
	}}
	if got := studioInstallerURL(rel, "2.64.1"); got != "https://example.test/a" {
		t.Fatalf("url = %q", got)
	}
	if got := studioInstallerURL(studioRelease{}, "2.64.1"); got != "" {
		t.Fatalf("missing asset must answer empty, got %q", got)
	}
}

// The running version an upstream install declares stays the panel's "current";
// the catalog's answer marks it and decides newer against the fork's releases.
func TestStudioCatalogMarksTheInstallVersionCurrent(t *testing.T) {
	srv := studioFake(t, `[{"tag_name":"studio-v2.64.0","published_at":"2026-11-15T00:00:00Z"}]`)
	withStudioRepo(t, srv)
	t.Cleanup(resetStudioCatalogCache)

	var hub update.VersionHub
	hub = studioCatalog(context.Background(), "2.64.0", true)
	if !hub.Versions[0].Current || hub.Newer {
		t.Fatalf("hub = %+v", hub)
	}
}
