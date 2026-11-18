package serve

import (
	"net/http"
	"sort"
	"strings"

	"reasonix/internal/session/control"
	"reasonix/internal/state/sessionstore"
)

// DispatchBoardParty is one end of a dispatch: the session that handed a task
// out or the one that received it. Titles and projects come from the tree join
// and degrade to the branch id when a session has gone quiet.
type DispatchBoardParty struct {
	Key         string `json:"key,omitempty"`
	SessionPath string `json:"sessionPath,omitempty"`
	Title       string `json:"title,omitempty"`
	Project     string `json:"project,omitempty"`
}

// DispatchBoardRow is one task handed from one session to another, with the
// state the dispatcher can read (waiting / working / replied / declined).
// It carryies no task text: the audit trail never stores message bodies.
type DispatchBoardRow struct {
	Dispatcher DispatchBoardParty `json:"dispatcher"`
	Assignee   DispatchBoardParty `json:"assignee"`
	Role       string             `json:"role,omitempty"`
	State      string             `json:"state"`
	Since      string             `json:"since,omitempty"`
	// CrossProject is true when the two ends live in different folders.
	CrossProject bool `json:"crossProject"`
}

// dispatchBoardHTTP answers the window's session-collaboration board: every
// dispatch this machine still has a record of, from both ends' point of view.
func (h *Hub) dispatchBoardHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, h.dispatchBoard())
}

func (h *Hub) dispatchBoard() []DispatchBoardRow {
	type partyMeta struct{ title, project string }
	meta := map[string]partyMeta{}
	paths := map[string]bool{}
	for _, ref := range h.sessionRoots() {
		rows, unlisted := h.treeSessionRows(ref.dir)
		titles := h.titleCacheFor(SessionDirFor(ref.dir)).snapshot()
		for _, row := range append(rows, unlisted...) {
			paths[row.Path] = true
			name := strings.TrimSuffix(sessionstore.BranchID(row.Path), ".jsonl")
			meta[control.PeerKeyForPath(row.Path)] = partyMeta{title: rowTitle(titles, row, name)(), project: projectOfRow(row, ref.dir)}
		}
	}

	// One row per (dispatcher, assignee) pair. The audit-backed status reader
	// already folds held/approved/replied/declined for one dispatcher, so we
	// reuse it instead of re-deriving states from the trail.
	seen := map[string]bool{}
	var out []DispatchBoardRow
	for path := range paths {
		for _, th := range h.dispatchThreadsStatus(path) {
			assigneeKey := th.SessionPath
			if assigneeKey == "" {
				continue
			}
			id := path + "\x00" + assigneeKey
			if seen[id] {
				continue
			}
			seen[id] = true
			dm := meta[control.PeerKeyForPath(path)]
			am := meta[control.PeerKeyForPath(th.SessionPath)]
			disp := DispatchBoardParty{Key: control.PeerKeyForPath(path), SessionPath: path, Title: dm.title, Project: dm.project}
			assign := DispatchBoardParty{Key: control.PeerKeyForPath(th.SessionPath), SessionPath: th.SessionPath, Title: th.TargetTitle, Project: th.TargetProject}
			if assign.Title == "" {
				assign.Title = am.title
			}
			if assign.Project == "" {
				assign.Project = am.project
			}
			out = append(out, DispatchBoardRow{
				Dispatcher:   disp,
				Assignee:     assign,
				Role:         th.Role,
				State:        string(th.State),
				Since:        th.Since.Format("2006-01-02T15:04:05Z07:00"),
				CrossProject: dm.project != "" && assign.Project != "" && !sameFolder(dm.project, assign.Project),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Since == out[j].Since {
			return out[i].Assignee.SessionPath < out[j].Assignee.SessionPath
		}
		return out[i].Since > out[j].Since
	})
	return out
}
