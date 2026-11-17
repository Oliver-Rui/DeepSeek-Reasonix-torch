import { forwardRef } from "react";
import { createPortal } from "react-dom";
import { t } from "../i18n";
import type { HubPort, TreeSession, TreeWorkspace } from "../port/hub";
import { StudioIcon } from "./StudioIcon";
import { WorkspaceReveal } from "./WorkspaceReveal";
import { workspaceMenuKeys } from "./menuKeys";

export interface ProjectMenuProps {
  ws: TreeWorkspace;
  visible: TreeSession[];
  searching: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onRename: () => void;
  onPin: () => void;
  onArchiveSessions: () => void;
  onBatchSelect: () => void;
  onRemove: () => void;
  hub: HubPort;
  dismiss: () => void;
  onError: (e: unknown) => void;
}

/** A project's actions, drawn at the document root so the sidebar's clipping
 *  cannot cut them. The move rows lead; every action closes the menu itself. */
export const ProjectMenu = forwardRef<HTMLDivElement, ProjectMenuProps>(function ProjectMenu(
  { ws, visible, searching, canMoveUp, canMoveDown, onMoveUp, onMoveDown, onRename, onPin, onArchiveSessions, onBatchSelect, onRemove, hub, dismiss, onError },
  ref,
) {
  return createPortal(
    <div ref={ref} className="session-pop" role="menu" aria-label={t("common.project_actions")} data-action-keydown="workspace.menu" data-target={ws.root} onKeyDown={workspaceMenuKeys} onClick={(ev) => ev.stopPropagation()}>
      <div className="session-pop-head">
        <b>{ws.name}</b>
        <small className="session-pop-path" title={ws.root}>{ws.root}</small>
      </div>
      <div className="session-pop-group">
        <button role="menuitem" data-action="workspace.move" data-value="up" disabled={searching || !canMoveUp} onClick={onMoveUp}>
          <StudioIcon name="up" /><span>{t("common.move_up")}</span>
        </button>
        <button role="menuitem" data-action="workspace.move" data-value="down" disabled={searching || !canMoveDown} onClick={onMoveDown}>
          <StudioIcon name="down" /><span>{t("common.move_down")}</span>
        </button>
      </div>
      <div className="session-pop-group">
        <button role="menuitem" data-action="workspace.rename-start" disabled={ws.missing} onClick={onRename}>
          <StudioIcon name="edit" /><span>{t("session.rename_project")}</span>
        </button>
        <button role="menuitem" data-action="workspace.pin" data-value={ws.pinned ? "off" : "on"} onClick={onPin}>
          <StudioIcon name="pin" /><span>{t(ws.pinned ? "session.unpin_project" : "session.pin_project")}</span>
        </button>
        <button role="menuitem" data-action="workspace.archive-sessions" disabled={visible.length === 0} onClick={onArchiveSessions}>
          <StudioIcon name="archive" /><span>{t("session.archive_project_sessions")}</span>
        </button>
      </div>
      <div className="session-pop-group">
        <button role="menuitem" data-action="session.selection" data-value="start" disabled={visible.length === 0} onClick={onBatchSelect}>
          <StudioIcon name="check" /><span>{t("session.batch_select_sessions")}</span>
        </button>
      </div>
      <WorkspaceReveal ws={ws} hub={hub} dismiss={dismiss} onError={onError} />
      <div className="session-pop-group">
        <button role="menuitem" data-action="workspace.copy-dir"
          onClick={() => { void navigator.clipboard.writeText(ws.root).catch(onError); dismiss(); }}>
          <StudioIcon name="copy" /><span>{t("session.copy_project_dir")}</span>
        </button>
      </div>
      <div className="session-pop-group">
        <button className="danger" role="menuitem" data-action="workspace.remove" onClick={onRemove}>
          <StudioIcon name="close" /><span>{t("common.remove_from_the_list")}</span><small>{t("common.keeps_files")}</small>
        </button>
      </div>
    </div>,
    document.body,
  );
});
