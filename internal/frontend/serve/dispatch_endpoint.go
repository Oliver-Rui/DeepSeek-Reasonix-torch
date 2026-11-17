package serve

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"reasonix/internal/base/nilutil"
	"reasonix/internal/session/control"
	"reasonix/internal/state/sessionstore"
)

// dispatchTarget is one target of a POST /tree/sessions/dispatch request.
type dispatchTarget struct {
	Mode         string `json:"mode"`
	SessionPath  string `json:"sessionPath,omitempty"`
	Project      string `json:"project,omitempty"`
	Role         string `json:"role"`
	Instructions string `json:"instructions,omitempty"`
	Text         string `json:"text"`
}

// dispatchRefusal is the per-target cause, carried as data so a frontend can
// phrase each failure without a translated top-level code for every one.
type dispatchRefusal struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// dispatchTargetResult answers one target: whether the task was handed off, and
// if not, which refusal stopped it.
type dispatchTargetResult struct {
	OK      bool             `json:"ok"`
	Refusal *dispatchRefusal `json:"refusal,omitempty"`
}

const dispatchMaxTargets = 20

// dispatchSessions hands a task to several targets at once. For mode "new" it
// reuses the model's dispatch gates and pairing; for "existing" it pairs the
// named, listed session and writes to its spool so an offline target picks the
// task up when it next ingests. Refusals are per target, never all-or-nothing,
// so one unknown path does not cancel the other tasks.
func (h *Hub) dispatchSessions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From    string           `json:"from"`
		Targets []dispatchTarget `json:"targets"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	from := strings.TrimSpace(body.From)
	fromRT := h.findSession(from)
	if fromRT == nil {
		refuse(w, http.StatusBadRequest, codeDispatchFromUnknown, "the sending session is not open here", nil)
		return
	}
	if _, ok := fromRT.Server.Controller().(*control.Controller); !ok {
		refuse(w, http.StatusBadRequest, codeDispatchFromUnknown, "the sending session is not local", nil)
		return
	}
	if len(body.Targets) == 0 {
		badBody(w)
		return
	}
	if len(body.Targets) > dispatchMaxTargets {
		refuse(w, http.StatusBadRequest, codeDispatchLimit, "too many dispatch targets", map[string]any{"max": dispatchMaxTargets})
		return
	}

	listed := h.listedSessionPaths()
	seen := map[string]bool{}
	results := make([]dispatchTargetResult, 0, len(body.Targets))
	for _, target := range body.Targets {
		key := dispatchTargetKey(target)
		if seen[key] {
			results = append(results, dispatchFail(codeDispatchDuplicate, "a target was named more than once"))
			continue
		}
		seen[key] = true
		result, targetPath := h.dispatchOne(r.Context(), fromRT, listed, target)
		results = append(results, result)
		if result.OK && strings.TrimSpace(target.Role) != "" {
			h.autoNameDispatchTarget(r.Context(), targetPath, target)
		}
	}
	writeJSON(w, results)
}

func dispatchTargetKey(t dispatchTarget) string {
	return t.Mode + "\x00" + strings.TrimSpace(t.SessionPath) + "\x00" + strings.TrimSpace(t.Project) + "\x00" + strings.TrimSpace(t.Role)
}

func dispatchFail(code, message string) dispatchTargetResult {
	return dispatchTargetResult{Refusal: &dispatchRefusal{Code: code, Message: message}}
}

func dispatchErrorRefusal(err error) dispatchTargetResult {
	switch {
	case errors.Is(err, control.ErrDispatchSameProject):
		return dispatchFail("session.dispatch_self", "a dispatch cannot open a session in the sender's own project")
	case errors.Is(err, control.ErrPeerSelf):
		return dispatchFail("session.dispatch_self", "a session cannot dispatch to itself")
	case errors.Is(err, control.ErrDispatchUnknownProject), errors.Is(err, control.ErrDispatchAmbiguous):
		return dispatchFail("project.unknown", "no open project answers to that name")
	case errors.Is(err, control.ErrDispatchUntrusted):
		return dispatchFail("dispatch.untrusted", "the user declined to trust that project")
	case errors.Is(err, control.ErrDispatchAwaitingTrust):
		return dispatchFail("dispatch.awaiting_trust", "the project's trust question is still open")
	case errors.Is(err, control.ErrDispatchCrossProject):
		return dispatchFail("dispatch.cross_project", "the user's cross-project setting refuses that task")
	case errors.Is(err, control.ErrDispatchLimit):
		return dispatchFail("dispatch.limit", "the session already has as many dispatched sessions open as the ceiling allows")
	case errors.Is(err, control.ErrDispatchDisabled):
		return dispatchFail("dispatch.disabled", "dispatch between sessions is disabled")
	case errors.Is(err, control.ErrPeersDisabled):
		return dispatchFail("peer.disabled", "messages between sessions are not enabled")
	case errors.Is(err, control.ErrPeerNotFound):
		return dispatchFail("session.dispatch_target_unknown", "the named session is not known here")
	default:
		return dispatchFail("dispatch.failed", err.Error())
	}
}

func (h *Hub) dispatchOne(ctx context.Context, fromRT *Runtime, listed map[string]string, target dispatchTarget) (dispatchTargetResult, string) {
	from := fromRT.Server.Controller().(*control.Controller)
	text := strings.TrimSpace(target.Text)
	if target.Instructions != "" {
		text = strings.TrimSpace(target.Instructions) + "\n\n" + text
	}
	if text == "" {
		return dispatchFail(codeMissingField, "text"), ""
	}
	switch target.Mode {
	case "new":
		project := strings.TrimSpace(target.Project)
		if project == "" {
			return dispatchFail(codeMissingField, "project"), ""
		}
		result, err := h.dispatchTo(ctx, fromRT, project, text, strings.TrimSpace(target.Role))
		if err != nil {
			return dispatchErrorRefusal(err), ""
		}
		targetPath := ""
		if rt := h.runtimeByKey(result.Key); rt != nil {
			targetPath = rt.Server.Controller().SessionPath()
		}
		return dispatchTargetResult{OK: true}, targetPath
	case "existing":
		path, err := filepath.Abs(strings.TrimSpace(target.SessionPath))
		if err != nil || path == "" {
			return dispatchFail(codeDispatchTargetUnknown, "the session path could not be resolved"), ""
		}
		listedPath, ok := listed[sessionstore.CanonicalSessionPath(path)]
		if !ok {
			return dispatchFail(codeDispatchTargetUnknown, "the named session is not listed here"), ""
		}
		key := control.PeerKeyForPath(listedPath)
		if key == from.PeerKey() {
			return dispatchFail(codeDispatchSelf, "a session cannot dispatch to itself"), ""
		}
		secret, err := control.NewPairSecret()
		if err != nil {
			return dispatchFail("dispatch.failed", err.Error()), ""
		}
		if rt := h.findSession(listedPath); rt != nil {
			if targetCtrl, ok := rt.Server.Controller().(*control.Controller); ok && targetCtrl.PeerKey() == key {
				targetCtrl.PairWithDispatcher(from.PeerKey(), secret)
			}
		}
		if role := strings.TrimSpace(target.Role); role != "" {
			if err := sessionstore.SetBranchRoleText(listedPath, role); err != nil {
				return dispatchFail("dispatch.failed", err.Error()), ""
			}
		}
		receipt, err := from.DispatchToSessionKey(key, strings.TrimSpace(target.Role), text, secret)
		if err != nil {
			return dispatchErrorRefusal(err), ""
		}
		h.dispatch.noteThread(from.SessionPath(), strings.TrimSpace(target.Role), listedPath, receipt.To.Key)
		return dispatchTargetResult{OK: true}, listedPath
	default:
		return dispatchFail(codeBadValue, "mode"), ""
	}
}

// autoNameDispatchTarget names a dispatched session from the task text and its
// role. A title model that is missing or fails leaves the role as the name,
// which is the fallback a dispatch has always had.
func (h *Hub) autoNameDispatchTarget(ctx context.Context, targetPath string, target dispatchTarget) {
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return
	}
	role := strings.TrimSpace(target.Role)
	root := h.sessionTreeRoot(filepath.Dir(targetPath))
	dir := filepath.Dir(targetPath)
	generator := h.sessionTitleGenerator(dir)
	if nilutil.IsNil(generator.currentTitleProvider().prov) {
		h.fallbackDispatchName(targetPath, root, role)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, titleFillTimeout)
	defer cancel()
	title, err := generator.tryGenerateGuidedTitle(ctx, target.Text, role)
	if err != nil || strings.TrimSpace(title) == "" {
		h.fallbackDispatchName(targetPath, root, role)
		return
	}
	cache := h.titleCacheFor(dir)
	if err := cache.putExplicit(filepath.Base(targetPath), strings.TrimSpace(title)); err != nil {
		slog.Warn("dispatch: could not cache the generated title", "err", err)
		h.fallbackDispatchName(targetPath, root, role)
		return
	}
	h.treeChanged()
}

// fallbackDispatchName applies the role-as-title fallback when guidance naming
// is unavailable. It needs a controller for nameDispatchPaneAfterRole; when the
// target has no live pane the sidecar name stands on its own.
func (h *Hub) fallbackDispatchName(targetPath, root, role string) {
	if role == "" {
		return
	}
	if rt := h.findSession(targetPath); rt != nil {
		if ctrl, ok := rt.Server.Controller().(*control.Controller); ok {
			h.nameDispatchPaneAfterRole(ctrl, root, role)
			return
		}
	}
	if err := sessionstore.RenameSession(targetPath, role); err != nil {
		slog.Warn("dispatch: could not fall back to the role title", "err", err)
	}
}

// runtimeByKey finds the local pane whose peer key matches.
func (h *Hub) runtimeByKey(key string) *Runtime {
	if key == "" {
		return nil
	}
	for _, rt := range h.localRuntimes() {
		if c, ok := rt.Server.Controller().(*control.Controller); ok && c.PeerKey() == key {
			return rt
		}
	}
	return nil
}
