package serve

import (
	"cmp"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"reasonix/internal/state/sessionstore"
)

const (
	rankStep = 1024
	// rankGap is the room a midpoint needs before the list is renumbered.
	rankGap = 1e-3
)

// compareSessions is the order a project's conversations are listed in: the ones
// never placed by hand first, newest activity first, so a new conversation
// lands on top as it always has; then the placed ones, by the rank the person
// gave them.
func compareSessions(a, b sessionstore.SessionInfo) int {
	switch {
	case a.ManualRank == 0 && b.ManualRank != 0:
		return -1
	case a.ManualRank != 0 && b.ManualRank == 0:
		return 1
	case a.ManualRank != 0:
		if c := cmp.Compare(a.ManualRank, b.ManualRank); c != 0 {
			return c
		}
	default:
		if c := b.LastActivityAt.Compare(a.LastActivityAt); c != 0 {
			return c
		}
	}
	return cmp.Compare(a.Path, b.Path)
}

func sortSessions(list []sessionstore.SessionInfo) { slices.SortFunc(list, compareSessions) }

// sessionPartition is the list a move may run inside: same archived state and
// same pin state. Pinned rows lead the sidebar, so a rank crossing that line
// could only be undone by the next read.
func sessionPartition(listed []sessionstore.SessionInfo, moved sessionstore.SessionInfo) []sessionstore.SessionInfo {
	out := make([]sessionstore.SessionInfo, 0, len(listed))
	for _, s := range listed {
		if s.Archived == moved.Archived && s.Pinned == moved.Pinned {
			out = append(out, s)
		}
	}
	return out
}

// rankUpdatesForOrder places group[at] and answers the ranks that change. While
// every row is placed and there is room, only the moved row changes; otherwise
// the partition is numbered in its new order.
func rankUpdatesForOrder(group []sessionstore.SessionInfo, at int) (map[string]float64, bool) {
	if len(group) < 2 {
		return nil, false
	}
	if updates, ok := midpointRank(group, at); ok {
		return updates, true
	}
	updates := make(map[string]float64, len(group))
	for i, s := range group {
		if rank := float64(i+1) * rankStep; rank != s.ManualRank {
			updates[s.Path] = rank
		}
	}
	return updates, true
}

// rankUpdatesForStep moves path one place up (-1) or down (1) inside its
// partition and answers the ranks that change.
func rankUpdatesForStep(listed []sessionstore.SessionInfo, path string, direction int) (map[string]float64, bool) {
	at := slices.IndexFunc(listed, func(s sessionstore.SessionInfo) bool { return s.Path == path })
	if at < 0 || (direction != -1 && direction != 1) {
		return nil, false
	}
	group := sessionPartition(listed, listed[at])
	from := slices.IndexFunc(group, func(s sessionstore.SessionInfo) bool { return s.Path == path })
	to := from + direction
	if from < 0 || to < 0 || to >= len(group) {
		return nil, false
	}
	moved := group[from]
	next := slices.Delete(slices.Clone(group), from, from+1)
	next = slices.Insert(next, to, moved)
	return rankUpdatesForOrder(next, to)
}

// rankUpdatesForPlace inserts path before before ("" for the end of its
// partition) and answers the ranks that change. The caller validates that
// before is a listed row in the same archived and pinned partition.
func rankUpdatesForPlace(listed []sessionstore.SessionInfo, path, before string) (map[string]float64, bool) {
	at := slices.IndexFunc(listed, func(s sessionstore.SessionInfo) bool { return s.Path == path })
	if at < 0 {
		return nil, false
	}
	group := sessionPartition(listed, listed[at])
	from := slices.IndexFunc(group, func(s sessionstore.SessionInfo) bool { return s.Path == path })
	if from < 0 {
		return nil, false
	}
	to := len(group)
	if before != "" {
		target := slices.IndexFunc(group, func(s sessionstore.SessionInfo) bool { return s.Path == before })
		if target < 0 || target == from {
			return nil, false
		}
		to = target
	}
	moved := group[from]
	next := slices.Delete(slices.Clone(group), from, from+1)
	insert := to
	if to > from {
		insert--
	}
	next = slices.Insert(next, insert, moved)
	if sameSessionOrder(next, group) {
		return nil, false
	}
	return rankUpdatesForOrder(next, insert)
}

func sameSessionOrder(a, b []sessionstore.SessionInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Path != b[i].Path {
			return false
		}
	}
	return true
}

// midpointRank ranks group[at] between its neighbours when all of the group is
// placed and the gap is wide enough.
func midpointRank(group []sessionstore.SessionInfo, at int) (map[string]float64, bool) {
	for _, s := range group {
		if s.ManualRank == 0 {
			return nil, false
		}
	}
	var rank float64
	switch {
	case at == 0:
		rank = group[1].ManualRank / 2
	case at == len(group)-1:
		rank = group[at-1].ManualRank + rankStep
	default:
		lo, hi := group[at-1].ManualRank, group[at+1].ManualRank
		if hi-lo < rankGap*2 {
			return nil, false
		}
		rank = (lo + hi) / 2
	}
	if rank < rankGap {
		return nil, false
	}
	return map[string]float64{group[at].Path: rank}, true
}

// moveSession places one conversation in its project's list. The rows are the
// same merged, grouped enumeration /tree answers, so a worktree project's rows
// move as one list. A before names a place; a direction names a step. A
// conversation a pane is driving has no transcript and answers without a rank.
func (h *Hub) moveSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path      string  `json:"path"`
		Direction int     `json:"direction"`
		Before    *string `json:"before"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	if body.Before == nil && body.Direction != -1 && body.Direction != 1 {
		badValue(w, "direction", "-1", "1")
		return
	}
	abs, err := filepath.Abs(strings.TrimSpace(body.Path))
	if err != nil {
		refuse(w, http.StatusBadRequest, codeSessionBadPath, "the session path could not be resolved", nil)
		return
	}
	root := h.sessionTreeRoot(abs)
	if root == "" {
		refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
		return
	}
	canonical := sessionstore.CanonicalSessionPath(abs)
	h.orderMu.Lock()
	defer h.orderMu.Unlock()
	listed, unlisted := h.treeSessionRowsForProject(h.sessionProjectRoots(root))
	path := ""
	for _, si := range listed {
		if sessionstore.CanonicalSessionPath(si.Path) == canonical {
			path = si.Path
			break
		}
	}
	if path == "" {
		for _, si := range unlisted {
			if sessionstore.CanonicalSessionPath(si.Path) == canonical {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
		return
	}
	if body.Before != nil {
		before := ""
		if raw := strings.TrimSpace(*body.Before); raw != "" {
			beforeAbs, err := filepath.Abs(raw)
			if err != nil {
				refuse(w, http.StatusBadRequest, codeSessionBadPath, "the target path could not be resolved", nil)
				return
			}
			beforeCanonical := sessionstore.CanonicalSessionPath(beforeAbs)
			for _, si := range listed {
				if sessionstore.CanonicalSessionPath(si.Path) == beforeCanonical {
					before = si.Path
					break
				}
			}
			if before == "" {
				refuse(w, http.StatusConflict, codeSessionOrderBoundary, "the target is not in this project's order", nil)
				return
			}
		}
		moved := listed[slices.IndexFunc(listed, func(s sessionstore.SessionInfo) bool { return s.Path == path })]
		if before != "" {
			target := listed[slices.IndexFunc(listed, func(s sessionstore.SessionInfo) bool { return s.Path == before })]
			if target.Archived != moved.Archived || target.Pinned != moved.Pinned {
				refuse(w, http.StatusConflict, codeSessionOrderBoundary, "a session cannot be placed across the pinned or archived boundary", nil)
				return
			}
		}
		updates, ok := rankUpdatesForPlace(listed, path, before)
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeSessionRanks(w, updates)
		return
	}
	updates, ok := rankUpdatesForStep(listed, path, body.Direction)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeSessionRanks(w, updates)
}

func writeSessionRanks(w http.ResponseWriter, updates map[string]float64) {
	paths := make([]string, 0, len(updates))
	for path := range updates {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	var failed error
	for _, path := range paths {
		failed = errors.Join(failed, sessionstore.SetSessionRank(path, updates[path]))
	}
	if failed != nil {
		writeErr(w, http.StatusInternalServerError, failed)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
