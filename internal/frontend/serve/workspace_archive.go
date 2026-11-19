package serve

import (
	"path/filepath"

	"reasonix/internal/state/sessionstore"
	"reasonix/internal/state/store"
)

// archiveWorkspaceSessions archives every conversation a folder holds, so a
// removed project leaves nothing behind in the everyday list. Nothing on disk is
// deleted: the rows are still in 归档, and adding the folder back restores them.
//
// A session that could not be archived is reported rather than skipped — half a
// removal is a state the person did not ask for.
func (h *Hub) archiveWorkspaceSessions(dir string) error {
	sessionDir := SessionDirFor(dir)
	if sessionDir == "" {
		return nil
	}
	infos, err := sessionstore.ListSessions(sessionDir)
	if err != nil {
		return err
	}
	open := h.openSessions()
	excluded := make([]string, 0, len(open))
	for openPath := range open {
		excluded = append(excluded, openPath)
	}
	for _, info := range infos {
		if info.Archived || store.IsSubagentTranscriptName(filepath.Base(info.Path)) {
			continue
		}
		if err := sessionstore.SetSessionLineageArchived(info.Path, true, excluded...); err != nil {
			return err
		}
	}
	return nil
}
