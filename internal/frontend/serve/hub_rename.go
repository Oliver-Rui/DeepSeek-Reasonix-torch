package serve

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"reasonix/internal/contract/tool"
	"reasonix/internal/session/control"
	"reasonix/internal/state/store"
)

var (
	errRenameUnknown  = errors.New("no session answers to that name; name one of your own sessions or a session you dispatched to")
	errRenameNotYours = errors.New("that session was not dispatched to by this one, so it cannot be renamed from here")
)

// renameSessionFor names the caller's own session (empty to) or one it handed
// a task to, after checking the target still lives in a workspace this hub
// owns. A name change writes the explicit title and refreshes the tree so the
// sidebar and the collaboration board pick it up at once.
func (h *Hub) renameSessionFor(from *Runtime, to, title string) (tool.RenamedSession, error) {
	ctrl, ok := from.Server.Controller().(*control.Controller)
	if !ok {
		return tool.RenamedSession{}, control.ErrRenameUnavailable
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return tool.RenamedSession{}, errors.New("a session name cannot be empty")
	}

	target, targetKind, err := h.resolveRenameTarget(from, ctrl, strings.TrimSpace(to))
	if err != nil {
		return tool.RenamedSession{}, err
	}

	path, err := h.resolveOwnedSessionPath(target)
	if err != nil {
		return tool.RenamedSession{}, err
	}
	if err := h.titleCacheFor(filepath.Dir(path)).putExplicit(filepath.Base(path), title); err != nil {
		return tool.RenamedSession{}, err
	}
	h.treeChanged()
	return tool.RenamedSession{Path: path, Title: title, Target: targetKind}, nil
}

// resolveRenameTarget turns what the model wrote into a session path and says
// whether it named itself ("self") or a dispatched member. Only these two are
// allowed: a key the caller did not dispatch to is refused, never guessed.
func (h *Hub) resolveRenameTarget(from *Runtime, ctrl *control.Controller, to string) (path, kind string, err error) {
	self := ctrl.SessionPath()
	selfKey := control.PeerKeyForPath(self)
	if to == "" || to == "self" || to == self || to == selfKey || strings.EqualFold(to, "自己") {
		if self == "" {
			return "", "", control.ErrRenameUnavailable
		}
		return self, "self", nil
	}

	// Only sessions this caller dispatched to are in scope. The durable thread
	// record (session paths) is the authority; the in-memory book is its mirror.
	allowed := map[string]bool{}
	for _, th := range h.dispatch.threadsFrom(self) {
		if th.SessionPath != "" {
			allowed[th.SessionPath] = true
		}
	}
	for _, id := range h.dispatch.targets(from.ID) {
		if rt := h.runtimeByID(id); rt != nil {
			if c, ok := rt.Server.Controller().(*control.Controller); ok && c.SessionPath() != "" {
				allowed[c.SessionPath()] = true
			}
		}
	}
	joins := h.dispatchTargetJoins()
	for candidate, j := range joins {
		if !allowed[j.sessionPath] {
			continue
		}
		if to == candidate || to == j.sessionPath || (j.title != "" && to == j.title) {
			return j.sessionPath, "member", nil
		}
	}
	for candidatePath := range allowed {
		if to == candidatePath || to == control.PeerKeyForPath(candidatePath) {
			return candidatePath, "member", nil
		}
	}
	// A key/title that resolves to a real session but was not dispatched to is a
	// distinct refusal from a name that resolves to nothing at all.
	for _, j := range joins {
		if to == j.title || to == j.sessionPath {
			return "", "", errRenameNotYours
		}
	}
	return "", "", errRenameUnknown
}

// resolveOwnedSessionPath validates a raw session path the same way the HTTP
// title routes do: a direct transcript file in a workspace this hub owns,
// with no symlink escape. It is the shared safety gate for both the routes and
// the rename tool.
func (h *Hub) resolveOwnedSessionPath(raw string) (string, error) {
	requested, err := filepath.Abs(strings.TrimSpace(raw))
	if err != nil {
		return "", errBadSessionPath
	}
	name := filepath.Base(requested)
	if !filepath.IsLocal(name) || !store.IsSessionTranscriptName(name) {
		return "", errBadSessionPath
	}
	for _, root := range h.sessionRoots() {
		dir, err := filepath.Abs(SessionDirFor(root.dir))
		if err != nil || !strings.HasPrefix(requested, dir+string(os.PathSeparator)) || filepath.Dir(requested) != dir {
			continue
		}
		path := filepath.Join(dir, name)
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		physicalDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(resolved, physicalDir+string(os.PathSeparator)) || filepath.Dir(resolved) != physicalDir {
			return "", errSessionOutside
		}
		return path, nil
	}
	return "", errSessionOutside
}

var (
	errBadSessionPath = errors.New("the session path could not be resolved")
	errSessionOutside = errors.New("path outside a known workspace")
)
