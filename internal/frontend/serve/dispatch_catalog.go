package serve

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"reasonix/internal/contract/config"
	"reasonix/internal/session/control"
	"reasonix/internal/state/peermail"
	"reasonix/internal/state/sessionstore"
)

// dispatchCatalogTTL is how long one directory walk serves every board and
// thread read. The board is a read model, not a ledger; a rename or archive
// invalidates it sooner, and a new dispatch target forces a rescan because a
// row the book knows and the catalog does not is a row the person just made.
const dispatchCatalogTTL = 15 * time.Second

type dispatchTargetJoin struct {
	title, project, sessionPath string
	archived, live              bool
}

// presentAndArchived re-reads the two facts that must not wait for the TTL:
// a deleted session is gone, and an archived one stays off the board. The
// sidecar read is bounded to the handful of sessions a dispatch actually has.
func (j dispatchTargetJoin) presentAndArchived() (present, archived bool) {
	if j.sessionPath == "" {
		return false, false
	}
	present = j.live
	if _, err := os.Stat(j.sessionPath); err == nil {
		present = true
	}
	archived = j.archived
	if meta, ok, err := sessionstore.LoadBranchMeta(j.sessionPath); err == nil && ok {
		archived = meta.Archived
	}
	return present, archived
}

type dispatchCatalogCache struct {
	mu      sync.Mutex
	at      time.Time
	entries map[string]dispatchTargetJoin
	epoch   atomic.Uint64
	builds  int
}

func (c *dispatchCatalogCache) invalidate() { c.epoch.Add(1) }

func (c *dispatchCatalogCache) buildCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.builds
}

// dispatchCatalog serves the one directory walk the board and the per-pane
// thread read share. Rebuilding under the mutex means ten panes asking at once
// pay for one walk, not ten.
func (h *Hub) dispatchCatalog(force bool) map[string]dispatchTargetJoin {
	h.board.mu.Lock()
	defer h.board.mu.Unlock()
	if !force && h.board.entries != nil && time.Since(h.board.at) < dispatchCatalogTTL {
		return h.board.entries
	}
	epoch := h.board.epoch.Load()
	entries := h.buildDispatchCatalog()
	h.board.builds++
	if force || epoch == h.board.epoch.Load() {
		h.board.entries, h.board.at = entries, time.Now()
	}
	return entries
}

func (h *Hub) buildDispatchCatalog() map[string]dispatchTargetJoin {
	out := map[string]dispatchTargetJoin{}
	seenDir := map[string]bool{}
	addDir := func(dir, projectRoot string) {
		key := sessionstore.CanonicalSessionPath(dir)
		if key == "" || seenDir[key] {
			return
		}
		seenDir[key] = true
		rows, unlisted := h.treeSessionRowsIn(dir, projectRoot)
		titles := h.titleCacheFor(dir).snapshot()
		add := func(row sessionstore.SessionInfo, live bool) {
			key := control.PeerKeyForPath(row.Path)
			if key == "" {
				return
			}
			name := strings.TrimSuffix(sessionstore.BranchID(row.Path), ".jsonl")
			out[key] = dispatchTargetJoin{
				title:       rowTitle(titles, row, name)(),
				project:     projectOfRow(row, projectRoot),
				sessionPath: row.Path,
				archived:    row.Archived,
				live:        live,
			}
		}
		for _, row := range rows {
			add(row, false)
		}
		for _, row := range unlisted {
			add(row, true)
		}
	}
	for _, ref := range h.sessionRoots() {
		addDir(SessionDirFor(ref.dir), ref.dir)
	}
	// A dispatch can reach a project this window has never opened, so the card
	// could not say what became of that target. Every project's session folder
	// is walked too; seenDir keeps it to one walk per folder.
	for _, dir := range projectSessionDirs() {
		addDir(dir, "")
	}
	return out
}

// projectSessionDirs lists every per-project session folder under the state
// root, whether or not this window has the project open. A dispatch is a
// durable record, so the session it names can be in a project the person has
// not looked at since; without this walk such a target would never be found and
// an archived or deleted one would keep its link forever.
func projectSessionDirs() []string {
	base := config.MemoryUserDir()
	if base == "" {
		return nil
	}
	projects := filepath.Join(base, "projects")
	entries, err := os.ReadDir(projects)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(projects, e.Name(), "sessions")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			out = append(out, dir)
		}
	}
	return out
}

type dispatchTrail struct {
	audit []peermail.AuditRecord
	joins map[string]dispatchTargetJoin
}

// loadDispatchAudit reads the durable trail once. The board reads it before it
// decides whether a catalog walk is worth doing at all.
func (h *Hub) loadDispatchAudit() []peermail.AuditRecord {
	dir := h.peersDir()
	if dir == "" {
		return nil
	}
	opened, err := peermail.Open(dir)
	if err != nil {
		return nil
	}
	return opened.AuditSince(time.Now().Add(-control.PeerThreadWindow))
}

// loadDispatchTrail reads the durable trail once for a request and attaches the
// cached catalog. The audit stays per-request so waiting/working/replied are
// current; only the directory walk is shared.
func (h *Hub) loadDispatchTrail(force bool) dispatchTrail {
	return dispatchTrail{audit: h.loadDispatchAudit(), joins: h.dispatchCatalog(force)}
}
