import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { t } from "../i18n";
import { reason, sayCode } from "../i18n/kernel";
import type { HubPort } from "../port/hub";
import type { DispatchResult, DispatchTarget } from "../port/dispatch";
import { StudioIcon } from "./StudioIcon";

type Row = {
  id: number;
  mode: "existing" | "new";
  sessionPath: string;
  project: string;
  role: string;
  instructions: string;
};

let nextId = 1;
const blankRow = (): Row => ({ id: nextId++, mode: "new", sessionPath: "", project: "", role: "", instructions: "" });

/** The human's side of a fan-out: one task, several targets, each with its own
 *  role. The kernel still owns the delivery — this only names where the task
 *  goes and shows which hand-offs it accepted. */
export function SessionDispatch({ from, projects, sessions, hub, onClose, onSent }: {
  from: { path: string; title: string };
  projects: { root: string; name: string }[];
  sessions: { path: string; title: string; project: string }[];
  hub: Pick<HubPort, "dispatch">;
  onClose: () => void;
  onSent: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const pending = useRef(false);
  const [text, setText] = useState("");
  const [rows, setRows] = useState<Row[]>([blankRow()]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [results, setResults] = useState<Map<number, DispatchResult> | null>(null);

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const el = dialog.current!;
    el.showModal();
    return () => { el.close(); previous?.focus(); };
  }, []);

  const patch = (id: number, next: Partial<Row>) =>
    setRows((current) => current.map((row) => row.id === id ? { ...row, ...next } : row));

  const send = async () => {
    if (pending.current) return;
    const targets: DispatchTarget[] = [];
    const sentRows: number[] = [];
    const body = text.trim();
    for (const row of rows) {
      if (!body) continue;
      if (row.mode === "existing" && !row.sessionPath) continue;
      if (row.mode === "new" && !row.project) continue;
      targets.push({
        mode: row.mode,
        sessionPath: row.mode === "existing" ? row.sessionPath : undefined,
        project: row.mode === "new" ? row.project : undefined,
        role: row.role.trim(),
        instructions: row.instructions.trim() || undefined,
        text: body,
      });
      sentRows.push(row.id);
    }
    if (!targets.length) return;
    pending.current = true;
    setBusy(true);
    setError("");
    try {
      const got = await hub.dispatch(from.path, targets);
      // Keyed by row id: a result belongs to the row that was sent, whatever
      // the list looks like after it.
      setResults(new Map(sentRows.map((row, i) => [row, got?.[i] ?? { ok: true }])));
    } catch (e) {
      setError(reason(e));
    } finally {
      pending.current = false;
      setBusy(false);
    }
  };

  const sent = results !== null && [...results.values()].every((r) => r.ok);

  return createPortal(
    <dialog ref={dialog} className="session-dispatch-dialog" aria-labelledby="session-dispatch-title"
      aria-busy={busy} data-action-cancel="layer.dismiss" data-action-keydown="layer.dismiss"
      onCancel={(ev) => { ev.preventDefault(); if (!busy) onClose(); }}
      onKeyDown={(ev) => { if (ev.key === "Escape") { ev.preventDefault(); ev.stopPropagation(); if (!busy) onClose(); } }}>
      <form data-action-submit="session.dispatch" data-target={from.path}
        onSubmit={(ev) => { ev.preventDefault(); void send(); }}>
        <header>
          <b id="session-dispatch-title">{t("chat.dispatch_task")} · {from.title}</b>
        </header>
        <label className="session-dispatch-field">
          <span>{t("chat.dispatch_task_text")}</span>
          <textarea value={text} rows={3} onChange={(ev) => setText(ev.target.value)}
            placeholder={t("chat.dispatch_task_placeholder")} disabled={busy || sent} />
        </label>
        <div className="session-dispatch-targets">
          {rows.map((row) => (
            <div className="session-dispatch-row" key={row.id} data-target={row.sessionPath || row.project}>
              <select aria-label={t("chat.dispatch_target_mode")} value={row.mode} disabled={busy || sent}
                data-action="session.dispatch-mode"
                onChange={(ev) => patch(row.id, { mode: ev.target.value as Row["mode"], sessionPath: "", project: "" })}>
                <option value="new">{t("chat.dispatch_new_session")}</option>
                <option value="existing">{t("chat.dispatch_existing_session")}</option>
              </select>
              {row.mode === "new" ? (
                <select aria-label={t("common.project")} value={row.project} disabled={busy || sent}
                  data-action="session.dispatch-project"
                  onChange={(ev) => patch(row.id, { project: ev.target.value })}>
                  <option value="">{t("chat.dispatch_pick_project")}</option>
                  {projects.map((p) => <option key={p.root} value={p.root}>{p.name}</option>)}
                </select>
              ) : (
                <select aria-label={t("chat.dispatch_existing_session")} value={row.sessionPath} disabled={busy || sent}
                  data-action="session.dispatch-target"
                  onChange={(ev) => patch(row.id, { sessionPath: ev.target.value })}>
                  <option value="">{t("chat.dispatch_pick_session")}</option>
                  {sessions.map((s) => <option key={s.path} value={s.path}>{s.title || s.path}</option>)}
                </select>
              )}
              <input aria-label={t("chat.dispatch_role")} value={row.role} disabled={busy || sent}
                data-action="session.dispatch-role" placeholder={t("chat.dispatch_role_placeholder")}
                onChange={(ev) => patch(row.id, { role: ev.target.value })} />
              <input aria-label={t("chat.dispatch_role_instructions")} value={row.instructions} disabled={busy || sent}
                data-action="session.dispatch-instructions" placeholder={t("chat.dispatch_role_instructions_placeholder")}
                onChange={(ev) => patch(row.id, { instructions: ev.target.value })} />
              {results && (() => {
                const r = results.get(row.id);
                return r ? <span className="session-dispatch-result" data-ok={r.ok ? "" : undefined}>{r.ok ? t("chat.dispatch_sent") : (r.refusal ? sayCode(r.refusal.code, r.refusal.message) : reason(r))}</span> : null;
              })()}
              {!busy && !sent && rows.length > 1 && (
                <button type="button" className="session-dispatch-remove" data-action="session.dispatch-remove"
                  aria-label={t("common.remove")} onClick={() => setRows((current) => current.filter((x) => x.id !== row.id))}>
                  <StudioIcon name="close" />
                </button>
              )}
            </div>
          ))}
        </div>
        {!busy && !sent && (
          <button type="button" className="session-dispatch-add" data-action="session.dispatch-add"
            onClick={() => setRows((current) => [...current, blankRow()])}>
            <StudioIcon name="plus" /><span>{t("chat.dispatch_add_target")}</span>
          </button>
        )}
        {error && <p className="session-dispatch-error" role="alert">{error}</p>}
        <footer>
          {sent ? (
            <button type="button" data-action="session.dispatch-done" onClick={onSent}>{t("common.done")}</button>
          ) : (
            <>
              <button type="button" data-action="layer.dismiss" disabled={busy} onClick={onClose}>{t("common.cancel")}</button>
              <button type="submit" data-action="session.dispatch" disabled={busy || !text.trim()}>{t("chat.dispatch_task")}</button>
            </>
          )}
        </footer>
      </form>
    </dialog>,
    document.body,
  );
}
