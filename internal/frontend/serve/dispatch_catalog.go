package serve

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	for _, ref := range h.sessionRoots() {
		rows, unlisted := h.treeSessionRows(ref.dir)
		titles := h.titleCacheFor(SessionDirFor(ref.dir)).snapshot()
		add := func(row sessionstore.SessionInfo, live bool) {
			key := control.PeerKeyForPath(row.Path)
			if key == "" {
				return
			}
			name := strings.TrimSuffix(sessionstore.BranchID(row.Path), ".jsonl")
			out[key] = dispatchTargetJoin{
				title:       rowTitle(titles, row, name)(),
				project:     projectOfRow(row, ref.dir),
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
