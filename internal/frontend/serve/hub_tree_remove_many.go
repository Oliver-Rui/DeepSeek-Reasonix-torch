package serve

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"reasonix/internal/state/sessionstore"
	"reasonix/internal/state/store"
)

// removeSessions deletes several conversations in one request. It reuses the
// single-delete safety: every path is validated and guarded before any file is
// erased, a running conversation refuses the whole batch, and a recovery copy
// covered by a parent is swept up with it.
func (h *Hub) removeSessions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil || len(body.Paths) == 0 {
		badBody(w)
		return
	}

	// Validate and resolve every requested path before touching anything, so a
	// malformed list never begins a delete that then aborts halfway.
	type picked struct {
		dir  string
		path string
	}
	leads := make([]picked, 0, len(body.Paths))
	seen := make(map[string]bool, len(body.Paths))
	open := h.openSessions()
	excluded := make([]string, 0, len(open))
	for p := range open {
		excluded = append(excluded, p)
	}
	for _, raw := range body.Paths {
		path, err := filepath.Abs(strings.TrimSpace(raw))
		if err != nil || !store.IsSessionTranscriptName(filepath.Base(path)) {
			refuse(w, http.StatusBadRequest, codeSessionBadPath, "a session path could not be resolved", nil)
			return
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		if id := open[sessionstore.CanonicalSessionPath(path)]; id != "" &&
			!h.releaseOrRefuse(w, r, "session.running", "a conversation is running; stop it first",
				h.panesWhere(func(rt *Runtime) bool { return rt.ID == id })) {
			return
		}
		dir := filepath.Dir(path)
		if !h.ownsSessionDir(dir) {
			refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
			return
		}
		leads = append(leads, picked{dir, path})
	}

	// Expand each lead to include recovery copies it covers, deduplicated across
	// the whole batch, then guard every transcript before erasing any.
	erase := make(map[string]string) // path -> dir
	for _, lead := range leads {
		paths := []string{lead.path}
		sibs, err := sessionstore.RecoveryLineagePaths(lead.path, excluded...)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		for _, p := range sibs {
			if p != lead.path && sessionstore.RecoveryBranchCoveredByParent(p, lead.dir) {
				paths = append(paths, p)
			}
		}
		for _, p := range paths {
			erase[p] = lead.dir
		}
	}
	ordered := make([]picked, 0, len(erase))
	for path, dir := range erase {
		ordered = append(ordered, picked{dir, path})
	}

	guards := make([]*sessionstore.SessionRemovalGuard, 0, len(ordered))
	for _, tgt := range ordered {
		guard, ok := acquireRemovalGuard(w, tgt.path)
		if !ok {
			for _, g := range guards {
				g.Release()
			}
			return
		}
		guards = append(guards, guard)
	}
	for i := range slices.Backward(ordered) {
		if !eraseGuardedSession(w, ordered[i].dir, ordered[i].path, guards[i]) {
			for _, g := range guards[:i] {
				g.Release()
			}
			return
		}
	}
	// A batch can empty a removed project's archive; each touched folder
	// retires its removal record the same way a single delete does.
	pruned := false
	for _, lead := range leads {
		if pruneRemovedRecordForSessionDir(lead.dir) {
			pruned = true
		}
	}
	if pruned {
		h.treeChanged()
	}
	w.WriteHeader(http.StatusNoContent)
}

// removeSession deletes a conversation the sidebar lists. A pane showing it is
// closed first, which lets the pane tear down its own jobs; one mid-turn is
// refused rather than pulled out from under the work.
func (h *Hub) removeSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	path, err := filepath.Abs(strings.TrimSpace(body.Path))
	if err != nil || !store.IsSessionTranscriptName(filepath.Base(path)) {
		refuse(w, http.StatusBadRequest, codeSessionBadPath, "the session path could not be resolved", nil)
		return
	}
	if id := h.openSessions()[sessionstore.CanonicalSessionPath(path)]; id != "" &&
		!h.releaseOrRefuse(w, r, "session.running", "this conversation is running; stop it first",
			h.panesWhere(func(rt *Runtime) bool { return rt.ID == id })) {
		return
	}
	dir := filepath.Dir(path)
	if !h.ownsSessionDir(dir) {
		refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
		return
	}
	// Removing only the lead lets a covered recovery copy take its place and the
	// delete reads as having done nothing. Copies holding content their parent
	// lacks, and any a pane has open, are left alone.
	open := h.openSessions()
	excluded := make([]string, 0, len(open))
	for openPath := range open {
		excluded = append(excluded, openPath)
	}
	paths := []string{path}
	sibs, err := sessionstore.RecoveryLineagePaths(path, excluded...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	for _, p := range sibs {
		if p != path && sessionstore.RecoveryBranchCoveredByParent(p, dir) {
			paths = append(paths, p)
		}
	}
	// Every guard is taken before anything is erased, so a held copy refuses the
	// request with the lead and the rest intact. The lead goes last.
	guards := make([]*sessionstore.SessionRemovalGuard, 0, len(paths))
	for _, p := range paths {
		guard, ok := acquireRemovalGuard(w, p)
		if !ok {
			for _, g := range guards {
				g.Release()
			}
			return
		}
		guards = append(guards, guard)
	}
	for i := range slices.Backward(paths) {
		if !eraseGuardedSession(w, dir, paths[i], guards[i]) {
			for _, g := range guards[:i] {
				g.Release()
			}
			return
		}
	}
	// An emptied archive retires its removal record: the ghost row and the
	// tombstone behind it go together, the way an emptied archive promised.
	if pruneRemovedRecordForSessionDir(dir) {
		h.treeChanged()
	}
	w.WriteHeader(http.StatusNoContent)
}

// acquireRemovalGuard answers the request itself when the session is held,
// returning false.
func acquireRemovalGuard(w http.ResponseWriter, path string) (*sessionstore.SessionRemovalGuard, bool) {
	// A pane's current path is narrower than "anyone writing this file": a
	// recovery branch or a mid-rotation session is held without being one.
	// Taking the guard beats probing it, which leaves a window for a writer.
	guard, err := sessionstore.TryAcquireSessionRemovalGuard(path)
	if err != nil {
		var held *sessionstore.SessionLeaseError
		if errors.As(err, &held) {
			if who := sessionHolder(held); who != nil {
				busy(w, "session.in_use_by", "another process holds this conversation open", who)
				return nil, false
			}
			busy(w, "session.in_use", "this conversation is still being written to", nil)
			return nil, false
		}
		writeErr(w, http.StatusInternalServerError, err)
		return nil, false
	}
	return guard, true
}

// eraseGuardedSession erases one transcript under its guard and answers the
// request itself when it cannot, returning false with the guard released.
func eraseGuardedSession(w http.ResponseWriter, dir, path string, guard *sessionstore.SessionRemovalGuard) bool {
	if err := removeSessionFiles(dir, path); err != nil {
		guard.Release()
		writeErr(w, http.StatusInternalServerError, err)
		return false
	}
	if err := guard.RemoveSidecarsAndRelease(); err != nil {
		// The conversation is already gone; a surviving lock file is stale
		// bookkeeping, not a failed delete.
		slog.Warn("serve: session removed, lock files survived", "path", path, "err", err)
	}
	return true
}

// sessionHolder names the process holding a conversation, and only when it is
// not this one. A lease this process holds means a write is in flight here,
// which is a different cause and carries a different code.
func sessionHolder(held *sessionstore.SessionLeaseError) map[string]any {
	if held == nil || held.Info == nil || held.Info.PID == 0 || held.Info.PID == os.Getpid() {
		return nil
	}
	return map[string]any{"pid": held.Info.PID, "host": held.Info.Hostname}
}
