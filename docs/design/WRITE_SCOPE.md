---
owner: @esengine
backup: @SivanCola
status: active
reviewed: 2026-11-24
---

# A write scope beside the approval mode

## The gap

Yolo today means "stop asking". It does not mean "may edit anything": the file writers refuse every
path outside `[sandbox] workspace_root`, `allow_write` and granted roots, and on Windows an enforcing
sandbox also holds bash inside a write-restricted token.

A user who picks 全部放行 and then asks an edit outside the project meets a refusal, and the note under
the picker ("approvals decide when to ask; the sandbox limits actual access") is the only thing that
explains why.

The menu row that pointed at the sandbox was removed because it drew a chevron and then left the menu:
it promised a submenu and delivered navigation. Removing it fixed the lie, not the need. The need is a
way to say "this session may write the whole disk", whose consequence is visible where it is chosen.

## What other agents do

| Product | Approval axis | Reach axis | Elevation path |
| --- | --- | --- | --- |
| Codex CLI | `approval_policy`: untrusted / on-failure / on-request / never | `sandbox_mode`: read-only / workspace-write / danger-full-access | Sandboxed command asks to run outside the sandbox; the answer is per command |
| Claude Code | modes: default / acceptEdits / plan / bypassPermissions, plus allow·ask·deny rules | Working directories, widened with `--add-dir`; optional OS sandbox for bash | `--dangerously-skip-permissions` skips prompts; a harmful-command guard still fires |
| ZCode | presets: plan / build / edit / yolo / guarded | Sandbox and network settings separate from the preset | `guarded` is yolo that still pauses on recognized destructive shell commands |
| Gemini CLI | `approvalMode`: default / autoEdit / yolo, plus tool allow/exclude lists | Sandbox backend: Docker / Seatbelt / Landlock | `--sandbox` composes with the mode; the two are chosen separately |
| Cline | Auto-approve toggled per action class | Command allow / deny lists | Each class opts in on its own |

Three things recur. Approval and reach are separate axes, and the products that make this easy to use
publish them as *presets* over both. Elevation is a named, one-time decision, never a silent consequence
of a mode name.

Every permissive setting still keeps a small set it will not cross: deny rules, the destructive-delete
guard, secrets, the harness's own state.

## The model

Four axes, each with its own owner. A posture is a named composition of them.

| Axis | Values | Owner |
| --- | --- | --- |
| Approval | `dontAsk` / `ask` / `auto` / `yolo` | `internal/session/control` (`ToolApproval*`) |
| Write scope | `workspace` (default) / `filesystem` | `[sandbox] write_scope`, `builtin.Workspace` |
| Execution | `enforce` / `off`, plus Windows `windows_integrity` | `[sandbox] bash`, `sandbox.Spec` |
| Network | on / off | `[sandbox] network` |

Rules (`deny` > `ask` > `allow`) are orthogonal to all four and keep precedence over every posture.

The write scope is the new axis. `workspace` is today's behaviour. `filesystem` removes the write-root
check from the file writers: any path is writable except the never-bypassable set below. Nothing else
changes — reads stay as they are, `forbid_read` still holds, the network setting still holds.

## What 全部放行 becomes

Choosing Yolo keeps asking one question, once: **只停止询问，保持工作区边界** (the default) or **停止询问，
并允许写入整块磁盘**. The answer is remembered per user, shown in the composer pill (`全部放行 · 全域可写`)
and written into the transcript, so a later reader of the session knows which authority it ran under.

The four menu entries stay four entries; the scope is not a fifth mode. Settings → Sandbox carries the
same choice for people who set it before a session starts.

The confirm dialog is the one place the consequence is stated in full: outside the project, every
existing file the account can write is reachable, and there is no undo beyond version control.

`docs/TOOL_APPROVAL_MODES.md` keeps saying Yolo does not bypass the sandbox; that sentence stays true for
the default answer. The full-disk answer is a different posture with its own name, not a reinterpretation
of Yolo.

## Headless

`reasonix run` has no confirm dialog, so it must not infer the full-disk answer from `--permission-mode
yolo`. The scope gets its own flag, `--write-scope workspace|filesystem`, defaulting to `workspace`.
A script that wants disk-wide writes says so; a script that upgraded Reasonix does not acquire them.

## The never-bypassable set

In every posture, including `filesystem`:

- Host secrets and credential files stay outside the writers' reach (`secrets.ForbiddenReadPaths`, the
  credential `.env` boundary).
- Reasonix's own state stays outside it: the config, the sandbox book, the schedule directory, remote
  state. An agent that can rewrite its own authority has escalated itself, and the next session would
  inherit that.
- `deny` rules, the recursive-delete guard, plan confirmation, ask questions and the external-text
  question keep their present force. Each is a decision, not a reach.

The rationale is a sentence per item, not a policy: these are the paths and decisions by which the
authority itself is constituted. Widening reach may not widen authority.

## Mechanics

`[sandbox] write_scope` takes `workspace` or `filesystem`; the desktop's posture is a session override,
the way `SandboxBashOverride` already works.

Under `filesystem`, `toolEnvironment.writeRoots` is built as "the filesystem minus the never-bypassable
set" instead of the workspace plus extras, and `Workspace.TargetAccessCheck` keeps the forbid and
host-secret checks it already runs.

On Windows an enforcing token bounds writes structurally, so the `filesystem` posture sets that session's
bash sandbox to `off` and says so in the confirm text; leaving it on would promise writes the token
refuses.

On macOS and Linux the OS sandbox is *widened* to the filesystem root for the session rather than
disabled, so its masking of the never-bypassable set still applies. Where a backend cannot express the
widening, the posture is refused with a reason instead of silently degrading to `off`.

## Gates

- Kernel: a `filesystem` session writes a file outside the project and is still refused `~/.ssh/id_rsa`,
  the config file and the schedule directory.
- Sandbox: the posture is refused, not degraded, when the backend cannot widen; the Windows path records
  the bash `off` it implied.
- Frontend: the Yolo confirm names both answers, the pill renders the chosen scope, and the menu still
  lists only the four modes.
- Docs: `TOOL_APPROVAL_MODES.md` gains the scope paragraph; the config template gains the key with the
  same wording as the confirm.

## Open decisions

- Whether `filesystem` should also widen reads. Recommendation: no — writing outside the project is a
  user's request to edit their files, and reading the whole disk is a different request with a different
  prompt.
- Whether the network setting should follow the scope. Recommendation: no — reach of the filesystem and
  reach of the network are separate consents.
- Whether Yolo should also pause on recognized destructive shell commands, as ZCode's `guarded` does.
  Recommendation: not in this change; the recursive-delete guard already refuses the worst class, and a
  phrase table for the rest is the kind of judgement this repository does not build.
