package serve

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"os"
	"path/filepath"
	"reasonix/internal/base/workspaceid"
	"reasonix/internal/state/sessionstore"
	"reasonix/internal/state/workspacelist"
	"strings"

	"reasonix/internal/base/backgroundwork"
	"reasonix/internal/base/fileutil"
	"reasonix/internal/contract/config"
	"reasonix/internal/contract/event"
	"reasonix/internal/platform/worktree"
	"reasonix/internal/state/migration"
	"reasonix/internal/state/store"
)

// maxRecoveryLineageWalk bounds the parent walk that groups recovery copies.
// The chain is short by construction; the bound is there so damaged metadata
// cannot spin the sidebar.
const maxRecoveryLineageWalk = 16

func (h *Hub) registerTreeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /tree", h.tree)
	mux.HandleFunc("GET /history/search", historySearch)
	mux.HandleFunc("GET /tree/events", h.treeEvents)
	mux.HandleFunc("GET /tree/events/replay", h.treeEventsReplay)
	mux.HandleFunc("POST /tree/workspaces/display-name", h.setWorkspaceDisplayName)
	mux.HandleFunc("POST /tree/workspaces/pin", h.setWorkspacePinned)
	mux.HandleFunc("POST /tree/workspaces", h.addWorkspace)
	mux.HandleFunc("POST /tree/workspaces/remove", h.removeWorkspace)
	mux.HandleFunc("POST /tree/workspaces/move", h.moveWorkspace)
	mux.HandleFunc("POST /tree/sessions/remove", h.removeSession)
	mux.HandleFunc("POST /tree/sessions/remove-many", h.removeSessions)
	mux.HandleFunc("POST /tree/sessions/archive", h.archiveSession)
	mux.HandleFunc("POST /tree/sessions/pin", h.pinSession)
	mux.HandleFunc("POST /tree/sessions/viewed", h.markSessionViewed)
	mux.HandleFunc("POST /tree/sessions/move", h.moveSession)
	mux.HandleFunc("POST /tree/sessions/move-to", h.moveSessionTo)
	mux.HandleFunc("POST /tree/sessions/pins", h.syncPins)
	mux.HandleFunc("POST /tree/sessions/rename", h.renameSession)
	mux.HandleFunc("POST /tree/sessions/auto-name", h.autoNameSession)
	mux.HandleFunc("POST /tree/sessions/export", h.exportSession)
	mux.HandleFunc("POST /tree/sessions/import-legacy", h.importLegacySessions)
	mux.HandleFunc("POST /tree/sessions/import-native", h.importNativeSession)
}

// tree answers the whole sidebar in one request: every remembered workspace
// with the sessions saved under it, and a folder only a pane was opened in once
// something was said there. Titles come from each project's cache and
// are never generated here — the pane that opens a session does that.
func (h *Hub) tree(w http.ResponseWriter, r *http.Request) {
	defer backgroundwork.Shared.Read()()
	window, ok := parseTreeWindow(w, r)
	if !ok {
		return
	}
	open := h.openSessions()
	roots := groupWorkspaces(h.roots())
	names := workspaceNames()
	pinned := workspacePinned()
	out := make([]treeWorkspace, 0, len(roots))
	for _, ref := range roots {
		// Open means a pane is driving this folder — true from the moment one is
		// opened, before its first turn has written a transcript to list.
		auto := isAutoWorkspace(ref.root)
		node := treeWorkspace{
			Root: ref.root, Name: fileutil.RootName(ref.root),
			DisplayName: names[ref.root],
			// A ghost is not pinned: a pin leads the list, and a removed project
			// has left the list — its stale pin must not drag the archive up.
			Pinned:         !ref.removed && pinned[projectIdentity(ref.root)],
			WorkspaceRoots: ref.roots,
			Isolated:       worktree.IsManagedPath(ref.root, config.DeliveryWorktreeDir()),
			Remembered:     ref.remembered,
			Removed:        ref.removed,
			Auto:           auto,
			Sessions:       []treeSession{},
		}
		for _, root := range ref.roots {
			node.Open = node.Open || h.rootPanes(root) > 0
		}
		if st, err := os.Stat(ref.root); err != nil || !st.IsDir() {
			node.Missing = true
		}
		page := h.projectPage(ref.roots, node.Name, open, window, window.limitFor(ref.root))
		// The 任务 group is a standing section, so the app's own folder is listed
		// even before anything was said in it — an unremembered project is not.
		if !ref.remembered && !auto && !page.Said && (page.Status == nil || page.Status.CountsComplete) {
			continue
		}
		// A ghost row stands where the archive is the view — the 已归档 scope,
		// and search. The everyday list stays clean unless live work still
		// happens in the removed folder: a pane's own row is not archive.
		if ref.removed && window.scope != scopeArchived && window.query == "" && !ghostIsLive(node, page.Sessions) {
			continue
		}
		node.Sessions, node.Total, node.Archived, node.Remaining = page.Sessions, page.Total, page.Archived, page.Remaining
		if page.Status != nil {
			node.ListingState, node.ListingRevision = page.Status.ListingState, page.Status.ListingRevision
			node.CountsComplete = &page.Status.CountsComplete
			node.PendingMetadata, node.FailedMetadata = page.Status.PendingMetadata, page.Status.FailedMetadata
		}
		out = append(out, node)
	}
	if window.recent {
		pageRecentTree(out, window.offset, window.limit)
	} else {
		h.reportOrphanPanes(out)
	}
	writeJSON(w, out)
}

// attachVersions hangs each conversation's earlier versions under its row. It
// runs after the rows exist because a version can be newer than its parent. A
// version a pane has open is left out: it is listed as that pane's own row.
func attachVersions(dir string, rows []treeSession, open map[string]string) {
	byParent, err := sessionstore.ListSessionVersions(dir)
	if err != nil || len(byParent) == 0 {
		return
	}
	for i := range rows {
		if filepath.Dir(rows[i].Path) != dir {
			continue
		}
		for _, v := range byParent[sessionstore.BranchID(rows[i].Path)] {
			if open[sessionstore.CanonicalSessionPath(v.Path)] != "" {
				continue
			}
			rows[i].Versions = append(rows[i].Versions, treeSession{
				Path: v.Path, Name: strings.TrimSuffix(filepath.Base(v.Path), ".jsonl"), Title: previewTitle(v.Preview), Turns: v.Turns,
			})
		}
	}
}

// unlistedOpenSessions are the open panes' conversations that have no file yet
// (a path is minted at the first submit, the transcript written after). A
// session on disk is never unlisted: a scope, search or window that leaves its
// row out is the view's choice.
func (h *Hub) unlistedOpenSessions(root string, shown []treeSession, onDisk []sessionstore.SessionInfo) []treeSession {
	runtimes := h.localRuntimes()
	// Transcript names are minted from a timestamp, so only a row named like an
	// open pane's file can be that pane's; resolving thousands of others costs
	// a filesystem walk each.
	names := make(map[string]bool, len(runtimes))
	for _, rt := range runtimes {
		if path := rt.Server.Controller().SessionPath(); path != "" {
			names[strings.ToLower(filepath.Base(path))] = true
		}
	}
	seen := make(map[string]bool, len(names)+len(shown))
	for _, si := range onDisk {
		if names[strings.ToLower(filepath.Base(si.Path))] {
			seen[sessionstore.CanonicalSessionPath(si.Path)] = true
		}
	}
	for _, row := range shown {
		seen[sessionstore.CanonicalSessionPath(row.Path)] = true
		for _, cp := range row.Copies {
			seen[sessionstore.CanonicalSessionPath(cp.Path)] = true
		}
	}
	var out []treeSession
	for _, rt := range runtimes {
		ctrl := rt.Server.Controller()
		path := ctrl.SessionPath()
		canonical := sessionstore.CanonicalSessionPath(path)
		if canonical == "" || seen[canonical] || !drivesRoot(rt, root) || !sessionstore.IsVisibleSession(path) {
			continue
		}
		seen[canonical] = true
		out = append(out, treeSession{
			Path: path, Name: strings.TrimSuffix(filepath.Base(path), ".jsonl"), RuntimeID: rt.ID,
		})
	}
	return out
}

// archiveSession changes catalog visibility without moving or deleting data.
// An idle pane on the session is closed first; one that is running is refused.
// Recovery siblings with another live pane stay active while the rest of the
// lineage is archived.
func (h *Hub) archiveSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path     string `json:"path"`
		Archived bool   `json:"archived"`
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
	if !h.ownsSessionDir(filepath.Dir(path)) {
		refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
		return
	}
	open := h.openSessions()
	excluded := make([]string, 0, len(open))
	for openPath := range open {
		excluded = append(excluded, openPath)
	}
	if err := sessionstore.SetSessionLineageArchived(path, body.Archived, excluded...); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pinSession records the user's "keep this" so the kernel's automatic archiving
// can honour it; the pin itself is only metadata and never blocks a run.
func (h *Hub) pinSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path   string `json:"path"`
		Pinned bool   `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	abs, err := filepath.Abs(strings.TrimSpace(body.Path))
	if err != nil {
		refuse(w, http.StatusBadRequest, codeSessionBadPath, "the session path could not be resolved", nil)
		return
	}
	path, ok := h.listedSessionPaths()[sessionstore.CanonicalSessionPath(abs)]
	if !ok {
		refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
		return
	}
	if err := sessionstore.SetSessionPinned(path, body.Pinned); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// syncPins adds the pins a window made before the kernel kept them. Until the
// first call the kernel will not archive anything on its own, because it
// cannot tell what the user pinned.
func (h *Hub) syncPins(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		badBody(w)
		return
	}
	listed := h.listedSessionPaths()
	var paths []string
	for _, raw := range body.Paths {
		abs, err := filepath.Abs(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		if path, ok := listed[sessionstore.CanonicalSessionPath(abs)]; ok {
			paths = append(paths, path)
		}
	}
	if err := sessionstore.PinSessions(paths); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	pinsSynced.Store(true)
	w.WriteHeader(http.StatusNoContent)
}

// listedSessionPaths maps each conversation the sidebar lists in a known
// workspace to the listed path, so a client-supplied path is only ever used
// through the value the kernel itself reported. The sidebar is the authority
// on what is listed: a conversation a pane is driving counts before its first
// transcript is on disk, and a trace or fork no row shows does not.
func (h *Hub) listedSessionPaths() map[string]string {
	out := map[string]string{}
	seen := map[string]bool{}
	for _, ref := range h.sessionRoots() {
		if seen[SessionDirFor(ref.dir)] {
			continue
		}
		seen[SessionDirFor(ref.dir)] = true
		listed, unlisted := h.treeSessionRows(ref.dir)
		for _, si := range listed {
			out[sessionstore.CanonicalSessionPath(si.Path)] = si.Path
		}
		for _, si := range unlisted {
			out[sessionstore.CanonicalSessionPath(si.Path)] = si.Path
		}
	}
	return out
}

// importLegacySessions is the user-facing recovery path for old installations.
// A conversation goes back to the project it was made in; one whose project is
// gone falls back to the workspace the user selected so it is not stranded outside
// the desktop tree, and one that records no project is left in the source and listed
// as skipped.
func (h *Hub) importLegacySessions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path      string `json:"path"`
		Workspace string `json:"workspace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	workspace, err := resolveWorkspaceDir(strings.TrimSpace(body.Workspace))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	known := ""
	for _, ref := range h.sessionRoots() {
		if ref.dir == workspace {
			known = ref.dir
			break
		}
	}
	if known == "" {
		refuse(w, http.StatusForbidden, "workspace.unknown", "select a known workspace for recovered sessions", nil)
		return
	}
	result := migration.RunLegacySessionImportInto(strings.TrimSpace(body.Path), SessionDirFor(known), legacyImportLog)
	for _, root := range result.Workspaces {
		// A folder the import routed sessions back into is a project the person
		// had in 1.x; without a place in the list its conversations are unreachable.
		if err := addRememberedWorkspace(r.Context(), root); err != nil && !errors.Is(err, errWorkspaceListFull) {
			slog.Warn("serve: could not list an imported project", "root", root, "err", err)
		}
	}
	count := 0
	for _, imported := range result.SessionImports {
		count += imported.Count
	}
	skipped := make([]legacySkipView, 0, len(result.SessionSkips))
	for _, skip := range result.SessionSkips {
		skipped = append(skipped, legacySkipView{Source: skip.Source, Name: skip.Name, Path: skip.Path, Reason: string(skip.Reason)})
	}
	writeJSON(w, legacyImportView{
		Summary: result.Summary(), Imported: count, Warnings: len(result.SessionErrs) + len(skipped),
		Recognised: !result.Unrecognised, Skipped: skipped,
	})
}

// legacySkipView is one 1.x session the import left in place. Reason is a
// sessionstore.SkipReason code; Path is the untouched entry in the source.
type legacySkipView struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type legacyImportView struct {
	Summary    string           `json:"summary"`
	Imported   int              `json:"imported"`
	Warnings   int              `json:"warnings"`
	Recognised bool             `json:"recognised"`
	Skipped    []legacySkipView `json:"skipped"`
}

var legacyImportLog = event.FuncSink(func(e event.Event) {
	if e.Level == event.LevelWarn {
		slog.Warn("serve: legacy import", "detail", e.Text)
	}
})

// recoveryLineageRoot names the conversation a copy belongs to. The stamped
// root is authoritative: walking parents instead splits one chain into a row
// per reclaimed middle link, which is precisely what GC leaves behind.
func recoveryLineageRoot(si sessionstore.SessionInfo, byID map[string]sessionstore.SessionInfo) string {
	if root := strings.TrimSpace(si.RecoveryRootID); root != "" {
		return root
	}
	id := sessionstore.BranchID(si.Path)
	for range maxRecoveryLineageWalk {
		info, ok := byID[id]
		if !ok || !info.Recovered {
			return id
		}
		parent := strings.TrimSpace(info.ParentID)
		if parent == "" || parent == id {
			return id
		}
		id = parent
	}
	return id
}

// addWorkspace remembers a folder without opening it. Adding and opening are
// separate acts: the sidebar lists what you work on, panes are what you drive.
// setWorkspaceDisplayName records the name a project reads by in the sidebar:
// the folder on disk keeps its own, and its files stay where they are.
func (h *Hub) setWorkspaceDisplayName(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	dir, err := resolveWorkspaceDir(strings.TrimSpace(body.Path))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	err = updateWorkspaceList(r.Context(), false, func(l *workspacelist.List) error {
		l.SetName(storedWorkspacePath(l, dir), strings.TrimSpace(body.Name))
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	h.treeChanged()
	writeJSON(w, map[string]any{"ok": true})
}

// setWorkspacePinned pins or unpins a project: a pinned one leads the list.
func (h *Hub) setWorkspacePinned(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path   string `json:"path"`
		Pinned bool   `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	dir, err := resolveWorkspaceDir(strings.TrimSpace(body.Path))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	err = updateWorkspaceList(r.Context(), false, func(l *workspacelist.List) error {
		l.SetPinned(storedWorkspacePath(l, dir), body.Pinned)
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	h.treeChanged()
	writeJSON(w, map[string]any{"ok": true})
}

// storedWorkspacePath names the path exactly as the list carries it, so a
// metadata write lands on the entry the sidebar reads.
func storedWorkspacePath(l *workspacelist.List, dir string) string {
	for _, stored := range l.Paths {
		if strings.EqualFold(strings.TrimSpace(stored), strings.TrimSpace(dir)) {
			return stored
		}
	}
	return dir
}

func (h *Hub) addWorkspace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	dir, err := resolveWorkspaceDir(strings.TrimSpace(body.Path))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	// A named add makes the folder first: a blank project is an empty, chosen
	// folder inside the one the picker returned, never a rename of it.
	if name := strings.TrimSpace(body.Name); name != "" {
		if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
			refuse(w, http.StatusBadRequest, "workspace.bad_name", "that name cannot be a folder", nil)
			return
		}
		created := filepath.Join(dir, name)
		if st, statErr := os.Stat(created); statErr != nil {
			if os.Mkdir(created, 0o755) != nil {
				writeErr(w, http.StatusInternalServerError, errors.New("could not create the folder"))
				return
			}
		} else if !st.IsDir() {
			refuse(w, http.StatusConflict, "workspace.name_taken", "a file with that name is already there", nil)
			return
		}
		dir = created
	}
	if err := addRememberedWorkspace(r.Context(), dir); err != nil {
		if errors.Is(err, errWorkspaceListFull) {
			refuse(w, http.StatusConflict, "workspace.limit_reached", err.Error(), nil)
		} else {
			writeErr(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, treeWorkspace{Root: dir, Name: fileutil.RootName(dir), Sessions: []treeSession{}})
}

// removeWorkspace drops a folder from the sidebar and archives what it held, so
// the conversations go with it — archived, not deleted, still found in 归档, and
// back again when the folder is added.
func (h *Hub) removeWorkspace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	dir := strings.TrimSpace(body.Path)
	if dir == "" {
		missingField(w, "path")
		return
	}
	inFolder := func(rt *Runtime) bool {
		return rt.Local() && projectIdentity(rt.Server.Controller().WorkspaceRoot()) == projectIdentity(dir)
	}
	if !h.releaseOrRefuse(w, r, "workspace.running", "a conversation in this folder is running; stop it first", h.panesWhere(inFolder)) {
		return
	}
	if err := h.archiveWorkspaceSessions(dir); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	forgetWorkspace(dir)
	// A folder whose archive the removal just emptied keeps no ghost row, so
	// the record it would stand for retires with the removal itself.
	pruneRemovedWorkspaceRecord(dir)
	h.treeChanged()
	w.WriteHeader(http.StatusNoContent)
}

// exportSession returns the transcript only after applying the same workspace
// boundary as rename and removal. The shell owns where the copy is saved; the
// kernel owns which path is safe to read.
func (h *Hub) exportSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badBody(w)
		return
	}
	path, err := filepath.Abs(strings.TrimSpace(body.Path))
	name := filepath.Base(path)
	if err != nil || !filepath.IsLocal(name) || !store.IsSessionTranscriptName(name) {
		refuse(w, http.StatusBadRequest, codeSessionBadPath, "the session path could not be resolved", nil)
		return
	}
	// The file's own directory is the opened root: task conversations live in
	// dated directories under the sessions tree, which the boundary check
	// proved owned. The validated local base name cannot leave it.
	dir := filepath.Dir(path)
	if !h.ownsSessionDir(dir) {
		refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
		return
	}
	path = filepath.Join(dir, name)
	root, err := os.OpenRoot(dir)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer root.Close()
	info, err := root.Lstat(name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !info.Mode().IsRegular() {
		refuse(w, http.StatusForbidden, "session.outside_workspace", "session export requires a regular transcript", nil)
		return
	}
	loaded, err := sessionstore.LoadSession(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	content, err := root.ReadFile(name)
	if loaded.Composition() != nil {
		content, err = sessionstore.ExportNativeArchive(path)
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Content: string(content)})
}

// ownsSessionDir reports whether dir sits inside a workspace's sessions tree
// this hub lists, which is the boundary a delete may not reach past.
func (h *Hub) ownsSessionDir(dir string) bool {
	return h.knownSessionDir(dir) != ""
}

// knownSessionDir reports which workspace's sessions tree dir belongs to: the
// sessions folder itself, or — since task conversations file themselves by
// date (…/sessions/2026/11/14/) — any directory inside that tree. The boundary
// a delete may not reach past is the tree, so everything under it is owned.
func (h *Hub) knownSessionDir(dir string) string {
	if root := h.sessionTreeRoot(dir); root != "" {
		return SessionDirFor(root)
	}
	return ""
}

// sessionTreeRoot names the workspace whose sessions tree holds path — the
// sessions folder itself or anything under it, dated task folders included —
// or "" when no listed workspace answers for it.
func (h *Hub) sessionTreeRoot(path string) string {
	for _, ref := range h.sessionRoots() {
		known, err := filepath.Abs(SessionDirFor(ref.dir))
		if err != nil {
			continue
		}
		if rel, err := filepath.Rel(known, path); err == nil && filepath.IsLocal(rel) {
			return ref.dir
		}
	}
	return ""
}

// rootRef is one sidebar folder, whether the user ever chose it, and whether
// it is a ghost: a folder the user removed that still holds conversations.
type rootRef struct {
	dir        string
	remembered bool
	removed    bool
}

// roots returns the sidebar's folders: everything remembered, plus whatever a
// pane is driving, so an open workspace can never be missing from the tree. A
// folder the user removed stays out of both until they ask for it again — but
// it keeps a ghost row for as long as its conversations exist, because a
// removal took the row off the sidebar, not the archive under it.
func (h *Hub) roots() []rootRef {
	paths := Workspaces()
	saved := map[string]bool{}
	for _, dir := range paths {
		if dir = strings.TrimSpace(dir); dir != "" {
			saved[dir] = true
		}
	}
	seen := map[string]bool{}
	var out []rootRef
	add := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		if workspaceRemoved(dir) {
			return
		}
		out = append(out, rootRef{dir: dir, remembered: saved[dir]})
	}
	for _, dir := range paths {
		add(dir)
	}
	for _, rt := range h.localRuntimes() {
		add(rt.Server.Controller().WorkspaceRoot())
	}
	// The app's own folder stands in the tree once it exists, pane or no pane:
	// the 任务 group holds its conversations, and closing a pane must not take
	// the written ones off the list.
	if auto := config.AutoWorkspaceDir(); auto != "" {
		if st, err := os.Stat(auto); err == nil && st.IsDir() {
			add(auto)
		}
	}
	out = append(out, ghostRoots()...)
	return out
}

// openSessions maps canonical session paths to the runtime driving them.
// drivesRoot reports whether rt is a local pane working in root.
func drivesRoot(rt *Runtime, root string) bool {
	return rt.Local() && sameFolder(rt.Server.Controller().WorkspaceRoot(), root)
}

// sameFolder says whether two spellings name one directory. The remembered
// folders and a pane's own root are written by different code, and on Windows a
// folder has several spellings (case, slash direction, a trailing separator, an
// 8.3 short name), so string equality splits one folder into two.
func sameFolder(a, b string) bool {
	if a == b {
		return true
	}
	fa := workspaceid.PathFingerprint(a)
	return fa != "" && fa == workspaceid.PathFingerprint(b)
}

// openSessionsIn is openSessions for the panes drivesRoot accepts: the ones
// unlistedOpenSessions will give a row of their own under root.
func (h *Hub) openSessionsIn(root string) map[string]string {
	out := map[string]string{}
	for _, rt := range h.localRuntimes() {
		if !drivesRoot(rt, root) {
			continue
		}
		if path := sessionstore.CanonicalSessionPath(rt.Server.Controller().SessionPath()); path != "" {
			out[path] = rt.ID
		}
	}
	return out
}

func (h *Hub) openSessions() map[string]string {
	out := map[string]string{}
	for _, rt := range h.localRuntimes() {
		if path := sessionstore.CanonicalSessionPath(rt.Server.Controller().SessionPath()); path != "" {
			out[path] = rt.ID
		}
	}
	return out
}

func (h *Hub) rootPanes(root string) int {
	n := 0
	for _, rt := range h.localRuntimes() {
		if sameFolder(rt.Server.Controller().WorkspaceRoot(), root) {
			n++
		}
	}
	return n
}

// titleCacheFor keeps one reader per project directory. Titles are file-name
// keyed and unique only within a project, so the caches must not be shared.
func (h *Hub) titleCacheFor(dir string) *titleCache {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.titles == nil {
		h.titles = map[string]*titleCache{}
	}
	if c := h.titles[dir]; c != nil {
		return c
	}
	c := newTitleCache(dir)
	h.titles[dir] = c
	return c
}

// workspaceRootForSession recovers which folder a transcript belongs to from
// its own sidecar, for an open request that names a session but no root.
func workspaceRootForSession(path string) string {
	meta, ok, err := sessionstore.LoadBranchMeta(strings.TrimSpace(path))
	if err != nil || !ok {
		return ""
	}
	return strings.TrimSpace(meta.WorkspaceRoot)
}

// decorateArchiveState reads the two marks the listing does not carry; the
// tree is bounded per workspace, so this is a handful of sidecar reads.
func decorateArchiveState(row *treeSession, path string) {
	m, ok, err := sessionstore.LoadBranchMeta(path)
	if err != nil || !ok {
		return
	}
	row.Pinned = m.Pinned
	if !m.AutoArchivedAt.IsZero() {
		row.AutoArchivedAt = m.AutoArchivedAt.UnixMilli()
	}
}
