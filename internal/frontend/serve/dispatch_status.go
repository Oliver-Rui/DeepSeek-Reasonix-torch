package serve

import (
	"net/http"
	"os"
	"strings"
	"time"

	"reasonix/internal/contract/config"
	"reasonix/internal/session/control"
	"reasonix/internal/state/sessionstore"
)

// dispatchThreadsHTTP answers the sidebar's record request at the hub. The
// per-pane route stays for a pane that asks its own server.
func (h *Hub) dispatchThreadsHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.dispatchThreadsStatus(r.URL.Query().Get("path")))
}

// DispatchThreadStatus is one target in a sender's dispatch record, as the
// sidebar's hover card reads it. State and Since come off the durable audit
// trail; Role comes from the in-memory book, and title and project from the
// hub's tree. A fresh dispatch whose message has not reached the trail yet is
// still reported from the book.
type DispatchThreadStatus struct {
	Role          string                  `json:"role,omitempty"`
	SessionPath   string                  `json:"sessionPath,omitempty"`
	TargetTitle   string                  `json:"targetTitle,omitempty"`
	TargetProject string                  `json:"targetProject,omitempty"`
	State         control.PeerThreadState `json:"state"`
	Since         time.Time               `json:"since"`
}

// dispatchThreadsStatus builds the enriched record for one sender path. It
// folds the sender's audit trail with the same helper the controller uses,
// joins each target's title and project from the tree, and fills anything the
// trail does not know from the book. Deriving the sender's key from its path
// makes the record survive a hub restart.
func (h *Hub) dispatchThreadsStatus(senderPath string) []DispatchThreadStatus {
	senderPath = strings.TrimSpace(senderPath)
	if senderPath == "" {
		return []DispatchThreadStatus{}
	}
	return h.dispatchThreadsFor(senderPath, h.loadDispatchTrail(false))
}

func (h *Hub) dispatchThreadsFor(senderPath string, trail dispatchTrail) []DispatchThreadStatus {
	self := control.PeerKeyForPath(senderPath)
	threads := control.PeerThreadsFrom(self, trail.audit)
	book := h.dispatch.threadsFrom(senderPath)
	join := trail.joins

	out := make([]DispatchThreadStatus, 0, len(threads)+len(book))
	seen := map[string]bool{}
	for _, th := range threads {
		row := DispatchThreadStatus{
			State: th.State, Since: th.Since,
			SessionPath: th.SessionPath, TargetTitle: th.Title, TargetProject: th.Project,
		}
		b, inBook := bookByKey(book, th.Key)
		if row.SessionPath == "" && inBook {
			row.SessionPath = b.SessionPath
		}
		// A target the person archived or deleted is a closed chapter: the trail
		// still records the message, so the card would keep a link that opens
		// nothing and names no session at all.
		if targetGone(join, th.Key, row.SessionPath) {
			seen[th.Key] = true
			continue
		}
		if j, ok := join[th.Key]; ok {
			if row.SessionPath == "" {
				row.SessionPath = j.sessionPath
			}
			if row.TargetTitle == "" {
				row.TargetTitle = j.title
			}
			if row.TargetProject == "" {
				row.TargetProject = j.project
			}
			if row.Role == "" && j.sessionPath != "" {
				row.Role = sessionstore.LoadBranchRoleText(j.sessionPath)
			}
		}
		if inBook {
			if row.Role == "" {
				row.Role = b.Role
			}
			if row.SessionPath == "" {
				row.SessionPath = b.SessionPath
			}
		}
		out = append(out, row)
		seen[th.Key] = true
	}
	for _, b := range book {
		key := b.Key
		if key == "" && b.SessionPath != "" {
			key = control.PeerKeyForPath(b.SessionPath)
		}
		if key == "" || seen[key] {
			continue
		}
		since := b.At
		if since.IsZero() {
			since = time.Now()
		}
		row := DispatchThreadStatus{
			Role: b.Role, SessionPath: b.SessionPath,
			State: control.PeerWaiting, Since: since,
		}
		if targetGone(join, key, row.SessionPath) {
			continue
		}
		if j, ok := join[key]; ok {
			row.TargetTitle, row.TargetProject = j.title, j.project
			if row.SessionPath == "" {
				row.SessionPath = j.sessionPath
			}
		}
		out = append(out, row)
		seen[key] = true
	}
	return out
}

// targetGone reports whether a dispatch target is no longer a session the card
// should offer a link to. The catalog answers when it knows the key; a path the
// book handed out answers when it does not, because a deleted file leaves the
// key in the trail and out of every listing. A key with no path anywhere stays:
// hiding it would take a live dispatch off the card.
func targetGone(join map[string]dispatchTargetJoin, key, knownPath string) bool {
	if j, ok := join[key]; ok {
		present, archived := j.presentAndArchived()
		return !present || archived
	}
	if knownPath == "" {
		return false
	}
	_, err := os.Stat(knownPath)
	return err != nil
}

// bookByKey finds a book entry by peer key, or by path when the key is empty.
func bookByKey(book []DispatchThread, key string) (DispatchThread, bool) {
	for _, b := range book {
		if b.Key == key || (key == "" && b.SessionPath == "") {
			return b, true
		}
	}
	return DispatchThread{}, false
}

// dispatchTargetJoins maps peer key to the title and project the tree knows for
// a session. Titles are read from the cached listing, never generated here.
func (h *Hub) dispatchTargetJoins() map[string]dispatchTargetJoin {
	return h.dispatchCatalog(false)
}

func projectOfRow(row sessionstore.SessionInfo, fallback string) string {
	if strings.TrimSpace(row.WorkspaceRoot) != "" {
		return row.WorkspaceRoot
	}
	return fallback
}

// peersDir names the durable spool root, resolved from this process's state
// root so it honours REASONIX_HOME the same way the controllers do.
func (h *Hub) peersDir() string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	return cfg.Roots().PeersDir()
}
