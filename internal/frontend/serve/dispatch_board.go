package serve

import (
	"net/http"
	"sort"
	"time"

	"reasonix/internal/session/control"
	"reasonix/internal/state/peermail"
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
func (h *Hub) dispatchBoardHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSONCached(w, r, h.dispatchBoard())
}

func (h *Hub) dispatchBoard() []DispatchBoardRow {
	audit := h.loadDispatchAudit()
	book := h.dispatch.snapshotThreads()
	if !hasDispatchSenders(audit, book) {
		return nil
	}
	trail := dispatchTrail{audit: audit, joins: h.dispatchCatalog(false)}
	senders := boardSenders(trail, book)
	if boardNeedsRescan(trail, book) {
		trail.joins = h.dispatchCatalog(true)
		senders = boardSenders(trail, book)
	}
	if len(senders) == 0 {
		return nil
	}

	seen := map[string]bool{}
	parties := map[string]struct {
		present, archived bool
	}{}
	party := func(key string) (dispatchTargetJoin, bool, bool) {
		join, ok := trail.joins[key]
		if !ok {
			return join, false, false
		}
		if state, ok := parties[key]; ok {
			return join, state.present, state.archived
		}
		present, archived := join.presentAndArchived()
		parties[key] = struct {
			present, archived bool
		}{present, archived}
		return join, present, archived
	}

	var out []DispatchBoardRow
	for senderKey, senderPath := range senders {
		dm, present, archived := party(senderKey)
		if !present || archived {
			continue
		}
		for _, th := range h.dispatchThreadsFor(senderPath, trail) {
			if th.SessionPath == "" {
				continue
			}
			targetKey := control.PeerKeyForPath(th.SessionPath)
			if targetKey == "" {
				continue
			}
			am, present, archived := party(targetKey)
			// Both ends must still exist and be live: an archived or deleted
			// session is a closed chapter, and must not keep asking for the
			// person's attention on the board.
			if !present || archived {
				continue
			}
			id := senderKey + "\x00" + targetKey
			if seen[id] {
				continue
			}
			seen[id] = true
			disp := DispatchBoardParty{Key: senderKey, SessionPath: senderPath, Title: dm.title, Project: dm.project}
			assign := DispatchBoardParty{Key: targetKey, SessionPath: th.SessionPath, Title: th.TargetTitle, Project: th.TargetProject}
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
	out = dedupeBoardReplies(out)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Since == out[j].Since {
			return out[i].Assignee.SessionPath < out[j].Assignee.SessionPath
		}
		return out[i].Since > out[j].Since
	})
	return out
}

// hasDispatchSenders is the cheap gate before the catalog walk: an idle machine
// with no dispatch record pays one audit read and nothing else.
func hasDispatchSenders(audit []peermail.AuditRecord, book map[string][]DispatchThread) bool {
	if len(book) > 0 {
		return true
	}
	for _, record := range audit {
		if record.Decision == peermail.DecisionSent {
			return true
		}
	}
	return false
}

// boardSenders is the set the board may show: the audit's senders that the
// catalog still knows, plus every sender the in-memory book remembers. A
// foreign sender that is not one of this machine's sessions is simply absent.
func boardSenders(trail dispatchTrail, book map[string][]DispatchThread) map[string]string {
	senders := map[string]string{}
	for _, record := range trail.audit {
		if record.Decision != peermail.DecisionSent || record.Sender == "" {
			continue
		}
		if join, ok := trail.joins[record.Sender]; ok && join.sessionPath != "" {
			senders[record.Sender] = join.sessionPath
		}
	}
	for senderPath := range book {
		key := control.PeerKeyForPath(senderPath)
		if join, ok := trail.joins[key]; ok && join.sessionPath != "" {
			senders[key] = join.sessionPath
		}
	}
	return senders
}

// boardNeedsRescan answers whether a recent book entry names a dispatch end the
// catalog does not. Only a recent thread can be a task the person just handed
// out; an old thread whose target was deleted must not force a walk forever.
func boardNeedsRescan(trail dispatchTrail, book map[string][]DispatchThread) bool {
	const recent = time.Minute
	for senderPath, threads := range book {
		fresh := false
		for _, thread := range threads {
			if thread.At.IsZero() || time.Since(thread.At) < recent {
				fresh = true
				break
			}
		}
		if !fresh {
			continue
		}
		if _, ok := trail.joins[control.PeerKeyForPath(senderPath)]; !ok {
			return true
		}
		for _, thread := range threads {
			if !thread.At.IsZero() && time.Since(thread.At) >= recent {
				continue
			}
			key := thread.Key
			if key == "" {
				key = control.PeerKeyForPath(thread.SessionPath)
			}
			if key == "" {
				continue
			}
			if _, ok := trail.joins[key]; !ok {
				return true
			}
		}
	}
	return false
}

// dedupeBoardReplies drops the role-less row that is a dispatch's reply: when
// the reverse direction carries a role, that row is the dispatch itself and
// carries the state lamp, so the reply must not appear as a second task.
func dedupeBoardReplies(rows []DispatchBoardRow) []DispatchBoardRow {
	kept := make([]DispatchBoardRow, 0, len(rows))
	for _, row := range rows {
		if row.Role == "" {
			reply := false
			for _, other := range rows {
				if other.Role != "" && other.Dispatcher.SessionPath == row.Assignee.SessionPath && other.Assignee.SessionPath == row.Dispatcher.SessionPath {
					reply = true
					break
				}
			}
			if reply {
				continue
			}
		}
		kept = append(kept, row)
	}
	return kept
}
