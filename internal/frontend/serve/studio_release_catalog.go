package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"reasonix/internal/platform/update"
)

// This fork publishes its releases on GitHub — no signed manifest on a mirror —
// so the version panel's catalog is the releases list itself, mapped onto the
// VersionHub shape the panel already renders, each row carrying its installer.
// Saving and running the file is the person's; self-apply stays upstream's.

// studioRepo is the repository whose releases are the catalog. A variable so a
// test can point it at a fake.
var studioRepo = "Oliver-Rui/DeepSeek-Reasonix-torch"

// studioAPIBase is the GitHub API root; a variable so a test points the
// catalog at a fake.
var studioAPIBase = "https://api.github.com"

// studioCatalogTTL keeps the panel and the background watch from spending the
// API's small unauthenticated budget on every look.
const studioCatalogTTL = 5 * time.Minute

type studioAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type studioRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt string        `json:"published_at"`
	Assets      []studioAsset `json:"assets"`
}

// studioCatalogCache holds the last good read, so the panel and the watch share
// one trip and a ?refresh=1 stays honest about bypassing it.
var studioCatalogCache struct {
	mu       sync.Mutex
	at       time.Time
	hub      update.VersionHub
	releases []studioRelease
}

// studioCatalog reads the fork's GitHub releases as the version catalog.
// current is the running build's version — empty on a build that cannot name
// itself, which the panel reads as a local build. Err rides beside the data:
// an unreachable GitHub must not hide which version is running.
func studioCatalog(ctx context.Context, current string, refresh bool) update.VersionHub {
	studioCatalogCache.mu.Lock()
	defer studioCatalogCache.mu.Unlock()
	fresh := time.Since(studioCatalogCache.at) < studioCatalogTTL
	if refresh || !fresh || studioCatalogCache.releases == nil {
		releases, err := fetchStudioReleases(ctx)
		if err != nil {
			if studioCatalogCache.releases != nil {
				// A stale catalog beats an empty one: keep answering the last
				// good read, stamped with why it is stale.
				hub := assembleStudioHub(current, studioCatalogCache.releases)
				hub.Err = err.Error()
				return hub
			}
			return update.VersionHub{Current: current, Err: err.Error()}
		}
		studioCatalogCache.releases = releases
		studioCatalogCache.at = time.Now()
	}
	return assembleStudioHub(current, studioCatalogCache.releases)
}

func fetchStudioReleases(ctx context.Context) ([]studioRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/releases?per_page=15", studioAPIBase, studioRepo), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "reasonix-studio-catalog")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github answered %d for the releases list", resp.StatusCode)
	}
	var releases []studioRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

// assembleStudioHub maps the releases onto the panel's shape: newest version
// first, the running one marked, installers carried beside their rows.
func assembleStudioHub(current string, releases []studioRelease) update.VersionHub {
	hub := update.VersionHub{Current: current, Studio: true}
	for _, rel := range releases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		version, ok := studioVersionOf(rel.TagName)
		if !ok {
			continue
		}
		entry := update.VersionEntry{
			Version:     version,
			Tag:         rel.TagName,
			PublishedAt: rel.PublishedAt,
			HasNotes:    strings.TrimSpace(rel.Body) != "",
			Current:     current != "" && version == current,
			Older:       current != "" && studioVersionLess(version, current),
			URL:         studioInstallerURL(rel, version),
		}
		hub.Versions = append(hub.Versions, entry)
	}
	sort.SliceStable(hub.Versions, func(i, j int) bool {
		return studioVersionLess(hub.Versions[j].Version, hub.Versions[i].Version)
	})
	if len(hub.Versions) > 0 {
		hub.Latest = hub.Versions[0].Version
		hub.Newer = current != "" && studioVersionLess(current, hub.Latest)
	}
	return hub
}

// studioVersionOf reads studio-vX.Y.Z (or vX.Y.Z) into the bare version.
func studioVersionOf(tag string) (string, bool) {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "studio-")
	tag = strings.TrimPrefix(tag, "v")
	parts := strings.Split(tag, ".")
	if len(parts) != 3 {
		return "", false
	}
	for _, part := range parts {
		if part == "" || len(part) > 6 {
			return "", false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return "", false
			}
		}
	}
	return tag, true
}

// studioVersionLess orders two dotted numeric versions left to right.
func studioVersionLess(a, b string) bool {
	as, aok := studioVersionParts(a)
	bs, bok := studioVersionParts(b)
	if !aok || !bok {
		return a < b
	}
	for i := range as {
		if as[i] != bs[i] {
			return as[i] < bs[i]
		}
	}
	return false
}

func studioVersionParts(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// studioInstallerURL picks the release's Windows installer asset, named with
// the version the release publishes.
func studioInstallerURL(rel studioRelease, version string) string {
	want := fmt.Sprintf("ReasonixStudio-%s-windows-amd64-installer.exe", version)
	for _, asset := range rel.Assets {
		if asset.Name == want && asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

// studioReleaseNotes answers one release's body from the catalog's read. A
// version the catalog does not carry answers nil: the caller falls back.
func studioReleaseNotes(ctx context.Context, version string, refresh bool) (string, bool) {
	studioCatalogCache.mu.Lock()
	defer studioCatalogCache.mu.Unlock()
	fresh := time.Since(studioCatalogCache.at) < studioCatalogTTL
	if refresh || !fresh || studioCatalogCache.releases == nil {
		releases, err := fetchStudioReleases(ctx)
		if err != nil {
			return "", false
		}
		studioCatalogCache.releases = releases
		studioCatalogCache.at = time.Now()
	}
	for _, rel := range studioCatalogCache.releases {
		if v, ok := studioVersionOf(rel.TagName); ok && v == version {
			return rel.Body, strings.TrimSpace(rel.Body) != ""
		}
	}
	return "", false
}
