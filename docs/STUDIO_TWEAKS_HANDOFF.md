---
owner: @Oliver-Rui
backup: @esengine
status: active
reviewed: 2026-11-15
---

# DeepSeek-Reasonix-torch — handoff

A personal fork of [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) on the
`studio` (2.x) line. This document is the state of the fork at its first pushed revision: what
changed, how it was checked, and what is still open.

## 1. What this revision changes

All paths are under `desktop/frontend-next/src` unless noted.

### Sidebar navigation, rebuilt as one list

- The icon rail and the open sidebar draw the **same rows from one table** (`ui/navdest.ts`
  `BAR` / `useBarEntries`). Before this, the two could disagree and one row could be offered
  twice; a duplicate 新建会话 row on the open sidebar is what prompted the change.
- `ui/SidebarDestinations.tsx` (new) renders those rows on the open column;
  `ui/NavMore.tsx` (new) holds the arrangement panel and the bar's own context menu.
- The panel opens **in two steps**: right-click the bar → a one-item menu (自定义导航栏) → the
  panel, which is centred with a scrim. One flat list, one row per entry, each row carrying a
  show switch and a drag handle. 新建会话 is always shown and cannot be switched off.
- Dragging reorders with a FLIP animation: the dragged row follows the pointer, crossed rows
  give way, and the order is persisted **on release only** — writing on every `pointermove` was
  what made it flicker.
- The rail's mode is now two answers, not three (`state/prefs.ts`): `collapsed` (shown while the
  workspace column is shut) and `off`. A stored `"on"` reads back as `collapsed`.
- The bar is separated from the project list by a hairline, matching the one above the settings
  row at the foot. The 项目 view name lost its pill.

### The conversation rail (right edge)

- Ticks are dense and right-aligned; only the left end moves, the way a ruler's ticks do. The
  viewport is one darker tick (`.srail-now`), not a painted band.
- No follow-the-pointer card. Pointing at a single message stretches **that tick** leftwards into
  a dark bar (`22×3px`, right edge fixed) and puts the message's own text in a card to its left
  (`.srail-card`), mirroring a left-edge list. Folded runs keep their own row list instead.
- `railgeom.grow()` and its neighbour magnification are gone: length is spent on the one place
  being read.

### 最近 ordering

- Ordering moved into the session filter menu, directly under 全部, as one row — not a toolbar
  button and not a switch.
- It cycles newest → oldest → unsorted, and the arrow on the right (↑ / ↓ / ⇅) always says
  which.
- A chosen scope is inked and ticked rather than painted as a block, and the menu's rows are
  the height of the conversation rows that opened it.

### Removed

- 社区与贡献者 and 赞助 left 关于: the QQ group, Discord, GitHub issues, the contributor list,
  and the PayPal, WeChat and Alipay QR codes.
- Every dependency went with them: `Community.tsx`, `Sponsor.tsx`, their tests,
  `contributors.json`, the three images, `sponsor.css`, `en_community.ts`, `en_sponsor.ts`,
  eight `community.*` / `sponsor.*` action entries, two settings anchors, and the sidebar's
  社区 entry.
- `communityLinks.ts` keeps only the maintainer, which the author line still shows.
- 抖音 (account id, copy button, QR code) — removed earlier in the same line of work.
- `RecentSessions.tsx` (the 最近 view) and its tests: 最近 is an ordering now, not a page.
- `Account.tsx`: after the sign-in card left the settings page, nothing in the product mounted
  it. It and its test are gone; the port methods stay, since the kernel still serves them.

### Session messages between projects (`internal/`)

- A dispatch may name the **role** of the session it creates (`DispatchToProject(ctx, project,
  text, role)`), so several sessions in one project do not read as one.
- New setting `peers.incoming`: what another of the user's sessions may send without a card —
  `ask`, `dispatch` (a paired session only, the default) or `always`. Only the asking is skipped;
  what the message starts still runs under this session's own approvals and sandbox.
- Sending by hand is gone from the UI: dispatch is the model's job, so the peers panel is the
  switches plus whatever is waiting for an answer.

## 2. How it was checked

Run from `desktop/frontend-next`:

| Check | Command | Result |
| --- | --- | --- |
| Types | `tsc --noEmit` | passes |
| Build | `vite build` | passes |
| UI census gate | `node tools/ui-census/main.mjs` | exit 0 — no orphaned action, no unregistered one |
| Census fixtures | `node tools/ui-census/verify.mjs` | `fixture reports identical across 10 outputs` |
| Focused tests | `vitest run src/actions.test.ts src/i18n/msg/msg.test.ts src/i18n/i18n.test.ts src/ui/AccountSettings.test.tsx src/ui/panel-dead-ends.interaction.test.tsx src/ui/Rail.test.tsx src/ui/railgeom.test.ts src/styles/railhit.test.ts src/ui/workspaces.test.ts src/ui/workspace-order.interaction.test.tsx src/ui/session-batch.interaction.test.tsx src/ui/nav-entries.interaction.test.tsx src/ui/nav-parity.interaction.test.tsx src/ui/navrail.interaction.test.tsx` | 14 files, 146 tests green |

Browser checks (Playwright + local Edge, `npm run dev` on :5273): the rail card sat 33px left of
its stretched tick with all tick right edges on one x; the filter menu measured 30px per row; the
fold-all button reported `lucide-chevrons-down-up` at 14px.

### Not verified

- **The full `vitest` suite** was not run here: it takes over ten minutes on this machine and the
  harness times a single file at 0.4s but a cold full run much longer. CI runs it
  (`.github/workflows/studio.yml`).
- **Go-side tests** were not run — no Go toolchain on this machine. CI's `ci.yml` covers the
  kernel: `gofmt`, lint, race, test.
- Dark theme was not screenshotted.

## 3. Publishing

CI is already in the repository; this fork did not add it.

| Workflow | Trigger | Covers |
| --- | --- | --- |
| `ci.yml` | push / PR to `studio` or `main-v2` | kernel: lint, race, test |
| `studio.yml` | push / PR to `studio` | `pnpm typecheck`, `pnpm test`, `pnpm census`, `pnpm census:verify`, `pnpm build` |
| `release-studio.yml` | push a `studio-v*` tag, or `workflow_dispatch` with an existing tag | packages, signs, publishes the GitHub release |
| `e2e-bot.yml` | comment `/e2e` on a pull request | end-to-end suite against the real provider |

**The branch matters.** `ci.yml` and `studio.yml` only run on `studio` (and `main-v2`), so work
on any other branch is unchecked until it lands there.

`release-studio.yml` expects credentials a fork does not inherit.

| Credential | Needed for | Without it |
| --- | --- | --- |
| `studio-release` environment | The publish job's approval gate | The release cannot publish |
| `APPLE_*` (five secrets) | Signing and notarizing macOS | macOS jobs cannot sign |
| minisign key | Signing every artifact | The publish job stops |
| R2 credentials | Mirroring and the Studio catalog | The release publishes, the catalog is not updated |
| Certum | Windows Authenticode, only when `vars.STUDIO_SIGNING_ENABLED` is `true` | Windows ships unsigned |

A candidate release can be published as a prerelease by using a pre-release version, which
also keeps it off the GitHub latest slot.

Note the version baseline: `release-notes/studio/` now carries `2.62.0.md`, while the highest
`studio-v*` tag on the remote is `studio-v2.50.0`. The first tag this fork pushes should
therefore be `studio-v2.62.0`.

## 4. Open items

- `Account`'s port methods (`accountLogin`, `accountPoll`, …) are still declared and still
  served by the kernel; only the UI is gone. If the sign-in card returns, restore
  `ui/Account.tsx` from git history rather than rewriting it.
- `SidebarTools.tsx` is reduced to the settings row; the hover card it used to draw was folded
  into `SidebarDestinations.tsx`.
- Nothing in the 任务 group idea from earlier discussions was implemented: the tree has no
  session that belongs to no workspace, so "非项目会话" has no data to stand on.

## 5. Revision 2.62.0

The first release of this line is **2.62.0**, and the notes now carry that name. Everything on
this page above describes the first revision; these six commits are what it took to make the
branch buildable and releasable as 2.62.0.

| Commit | Change |
| --- | --- |
| `e5591df` | Goldens refreshed to the dispatch schema in the tree (`role`, "one of the user's projects"). Before this the branch tip failed `internal/assembly/boot` on a difference the tree had already decided. |
| `3ff9d07` | `peers.incoming` decides what arrives without a card: `ask` holds, `dispatch` takes a paired session, `always` takes any. `SetPeersIncoming` refuses anything else, and the setting is writable from the settings sheet. |
| `08d63b8` | Cloud-backup row and the account/market search intents removed; 备份 lands on the local file backup. |
| `d788646` | A scope row's picker opens from the row's left edge instead of hanging off the right. |
| `1e1734d` | Six perf scripts reach 文件 through `openFiles`, as the ones fixed in `466dbfd` do. |
| `54b53eb` | Release notes renamed to `2.62.0.md`, upstream handover line, `peers.incoming` upgrade note. |

### Windows build on this machine

Installed 2026-11-13 at `%LOCALAPPDATA%\Programs\Reasonix Studio`; `logs/shell.log` records
`version=2.62.0 packaged=true platform=win32/x64` and a kernel handshake. The installer is
unsigned (`no code signing certificate configured`), so SmartScreen will warn on first run.

| Step | Command |
| --- | --- |
| SPA | `cd desktop/frontend-next && pnpm install --frozen-lockfile && pnpm build` |
| kernel | `PATH=<go>/bin:$PATH CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=v2.62.0" -o desktop/electron/bin/ ./cmd/reasonix-studio-host` |
| bundle | `pnpm exec electron-builder --dir -c.extraMetadata.version=2.62.0 -c.electronDist=<local electron-v45.2.3-win32-x64.zip>` |
| installer | `pnpm exec electron-builder --prepackaged ../../dist/win-unpacked --win nsis -c.extraMetadata.version=2.62.0` |

Three environment facts the CI recipe does not meet here:

- **No Go on PATH.** A portable go1.26.11 lives at `D:\Tools\gotool\go\bin`.
- **`build:host` cannot fetch the CUA driver.** Its GitHub fetch carries a 120s timeout and the
  line runs at ~40 KB/s. Fetch the lock file's artifact by hand, verify its sha256, and put
  `cua-driver.exe` in `desktop/electron/bin`.
- **electron-builder re-downloads Electron even when it is cached, and can hang for ten
  minutes.** Pass `-c.electronDist` pointing at a local zip.
- **Toolset downloads hit GitHub too.** Set
  `ELECTRON_BUILDER_BINARIES_MIRROR=https://npmmirror.com/mirrors/electron-builder-binaries/`,
  or pre-seed `<LOCALAPPDATA>/electron-builder/Cache/<release>/<archive>`.

Artifact: `dist/ReasonixStudio-windows-amd64-installer.exe`, sha256
`d4e12da2e193fc916ed81ff792d11c90b71f5544c6a2740e8fba52dbc02256a9`, 135 MB. It installs per user
(HKCU uninstall entry `Reasonix Studio 2.62.0`), and every PE file is amd64 except the
installer's own 32-bit `resources/elevate.exe`.

### Checks run for this revision

| Check | Result |
| --- | --- |
| `gofmt -l` on the changed Go files | clean |
| `go vet` on the affected packages | clean |
| `go run ./tools/repolint -changed` (what `make check` runs; `make` is not installed here) | clean, after shortening a six-line comment and a 362-column paragraph |
| `go test ./internal/contract/... ./internal/session/control/ ./internal/frontend/serve/` | pass |
| `vitest run` on the four touched test files | 47 pass |
| `pnpm typecheck` | clean |
| `go test ./internal/assembly/boot/` | the goldens pass; two tests fail on this machine (`TestEffectBundledSkillReferenceReachesProviderAfterCopyInstall`, `TestEffectChangedFileEditAfterTheReadIsKept`, the latter panicking on an empty request list). Both fail the same way on `HEAD` without these commits, so they are not this revision's. |

### Open items for this revision

- **`README.md` and `README.zh-CN.md` still describe a 2.6.x line** ("2.6.1, 2.6.2, 2.6.3 … no jump
  to 2.7"). The release is 2.62.0. One of the two has to move before a tag is pushed.
- `internal/contract/tool/peer.go` and `internal/session/control/peer_dispatch.go` still say
  "another of the user's projects" while the tool now accepts this session's own project when a
  role is given. The model-facing description is right; these two comments are not.
- The two failing boot tests above need a clean-machine run (CI on Linux) before a tag; they are
  the only red in the packages this revision touched.
- Only Windows was built here. `.deb`, `.rpm`, `.AppImage` and the macOS bundle still rely on
  `release-studio.yml` and its credentials.

## 6. Revision 2.63.0, folded into the 2.62.0 release

The work of this revision plus everything below shipped as the fork's **first** release,
`studio-v2.62.0` — an earlier `studio-v2.63.0` release was published, deleted the same day, and
its content folded forward; the version number was not reused.

Four commits first, all frontend (`desktop/frontend-next`), pushed to `studio` on
`Oliver-Rui/DeepSeek-Reasonix-continued` — the remote was retargeted this revision; the
`Reasonix-Studio-Tweaks` URL in older notes never existed on GitHub.

| Commit | Change |
| --- | --- |
| `1e4451d` | The session tree carries two explicit groups — 项目 over the remembered projects, 任务 over the non-remembered folder(s). The partition is a pure helper (`workspaceGroups.ts`); a list, not a scalar, because `roots()` can list several unremembered folders. |
| `75b11b9` | 新建会话 opens a 选择项目 card (search, local + remote projects, 创建项目, 不在项目中工作). The escape row sends **no root**, and `Hub.resolveRoot` falls back to the kernel's auto workspace (`<state>/workspaces/auto`) — the state-home path is never computed in the frontend. Per-project 新建 bypasses. |
| `d5436da` | Three layout fixes, each live-measured first: the settings windowbar's buttons opt out of the drag region (最小化/最大化/关闭 were dead while 设置 was open); the scope pill's chevron opts back to `inline-block` against the deliberate global `.studio-icon { display: block }`; scope dropdown rows declare their grid positions so an exception chip cannot auto-place into column one. |
| `17154c7` | `data.cloud_backup` — a key nothing had referenced since the feature left in 2.62.0's work. |

The README open item from §5 is closed: both READMEs now tell the 2.61.0 handover, the
`studio` line's first release 2.62.0, and point at this repository's releases page.

The four commits above were followed by five more before the release was cut:

| Commit | Change |
| --- | --- |
| `c6a50b4` | 会话互聊 kernel: `peers.cross_project` (always / paired / never, default always) gates dispatches into another project, and a dispatch with a role writes the role as the target's explicit title. `PeerThread` carries the role (book join by peer key). |
| `90fe800` | Settings gains a top-level 会话互聊 section — the folded `peers` section promoted, the three groups gathered, the two dispatch ceilings exposed, a 跨项目派发 seg, and the （实验） qualifier dropped. |
| `40335ab` | The dispatching pane grows a 会话互聊 bar — one row per outbound thread, live states, open affordance; the folded dispatch row shows 派发任务 · project · role. |
| `92d7453` | The sidebar's group heads become controls — 项目 ⌄ / 任务 ⌄ fold their own group, the folder/tasks icons and the toolbar's separate fold-all retire, project rows lose the folder icon. |
| `72886a4` | The READMEs' intro says what the branch changes now (three sections, removals stay in the release notes); `.gitignore` keeps ZCode's session plans out of repolint's full run. |

### Checks run for this revision

| Check | Result |
| --- | --- |
| `tsc --noEmit` | clean |
| `vitest run` (full) | 6788/6788 across 914 files; two files fail to load under the session sandbox (`?worker&url` transform denials on `pdfjs-dist`/`maplibre-gl`), unrelated to the diff |
| `npm run census` + `census:verify` | sealed; fixture reports identical |
| `repolint -changed` | clean |
| `vite build` | clean |
| The running app at 100% and 115% zoom | the picker, both groups, the pill, the dropdown rows and the windowbar verified; the windowbar's buttons also by computed `webkitAppRegion` with the platform attribute stubbed (the plain browser does not render them) |

### Released as studio-v2.62.0

- Release: [studio-v2.62.0](https://github.com/Oliver-Rui/DeepSeek-Reasonix-continued/releases/tag/studio-v2.62.0),
  merged to `studio` through PR #1 (merge commit `07441dd0`)
- An earlier [studio-v2.63.0](https://github.com/Oliver-Rui/DeepSeek-Reasonix-continued/releases/tag/studio-v2.63.0)
  was published from that head, deleted the same day, and the whole line republished as
  studio-v2.62.0 with the README intro and the 会话互聊 work — the version number was not reused
  for different content.
- Installer: `ReasonixStudio-windows-amd64-installer.exe`, unsigned — SmartScreen warns on first run
- Published by hand through the REST API: `release-studio.yml` cannot publish
  from a fork (minisign secrets, the `studio-release` environment and a prior
  tag are all absent)
- `git push` was unreachable from this machine; every commit, the merge and
  the release were built server-side through the Git Data API, and the
  API-rebuilt history is byte-exact — every tree and commit sha equals the
  local one

Two CI facts this revision leaves behind:

- The macOS legs of both workflows are pinned to `macos-15` after
  `macos-latest`'s arm64 pool refused to seat jobs for four runs in a row
  ("not acquired by Runner of type hosted")
- The five upstream `deploy-*` workflows are disabled on the fork — they
  deploy upstream's website and workers with upstream's credentials and can
  only fail here


## 7. Revision in progress — the session lands, then picks its project

This revision inverts the new-session flow to the reference design and finishes the sidebar's
two groups. Nothing below is pushed yet: the local commits sit on `feat/same-project-subagent`
waiting for the word to push, rerun CI and republish.

### Kernel (`internal/frontend/serve`, `internal/frontend/cli`)

- `Hub.resolveRoot` — a brand-new conversation (no root, no session path) no longer joins the
  first live runtime's folder: it opens in the app's own workspace (`ensureAutoWorkspace`).
  Restores and dispatches keep their explicit roots.
- `POST /workspace` accepts `{auto: true}` — the pill's 不使用项目 answer; the pane joins the
  same auto workspace. Deliberately unchanged: the switch carries no conversation continuity.
- `GET /tree` marks the auto workspace `auto: true` and keeps it listed with no pane on it
  (`roots()` stats the folder), so the 任务 group is a standing section, not a pane byproduct.
- `POST /tree/workspaces` accepts `name` — a blank project is an empty, chosen folder created
  inside the folder the picker returned, then remembered. Two new refusal codes
  (`workspace.name_taken`, `workspace.bad_name`) with frontend wording.
- `reasonix serve --workspace-switch` grants `POST /workspace` to a plain serve; the desktop
  window grants itself the same thing, so the browser UI can switch projects too.

### Frontend (`desktop/frontend-next`)

- The composer's project pill (`WorkspacePill.tsx`) — the ZCode-shaped palette: 搜索项目 search,
  project rows with the current marker, 添加新项目 submenu, 不使用项目.
- 新建空白项目 is a dialog (项目名称 + 源文件夹); 使用现有文件夹 keeps the picker.
- The pill sits above the composer card on its left edge: a task pane shows 进入项目工作, a
  project pane the project's name, and after 不使用项目 is confirmed the pill goes away until
  the pane works in a project again.
- Blank-project creation falls back to a typed path when no picker answers (`addws.ts` gained
  `pickFolder` and `addNamed`).
- The sidebar splits by the kernel's mark now: `workspaceGroups` puts `auto` workspaces in
  任务 and everything else in 项目; the auto node renders no folder row — its conversations
  hang directly under the 任务 head. Unremembered folders a pane drives show under 项目.
- The 先打开一个项目 hero is gone: which project a conversation belongs to is the pill's
  question, so the empty pane just introduces itself (`Hero.tsx`), and `needsProject`,
  `onKeepHere` and the `claimed` flag left the codebase.
- 新建会话 opens the pane directly (no picker modal); the picker's row data lives on as
  `pickerProjects` for the pill's palette.

### Checks run for this revision

| Check | Result |
| --- | --- |
| `tsc --noEmit` | clean |
| `vitest run src/ui src/i18n src/port` | pass, 809 files |
| `go test ./internal/frontend/serve/ -timeout 30m` | one machine-only failure (`TestSkillCatalogProjectsAuthorInvocationFlags/other_project`, the known repo-key-collision family, green on CI); the refusal-parity test passes after the two new codes were worded |
| `npm run census` + `census:verify` | clean; fixture reports identical |
| `repolint -changed` | clean after trimming the touched CSS/Go lines back under their budgets |
| The running app in the browser | the full flow: 新建会话 → 任务 group, the pill 进入项目工作 → palette (search, Reasonix Studio 当前, 添加新项目 submenu, 不使用项目), switching into a project and back, 不使用项目 hiding the pill, and a real task conversation run |

### The review round that followed the first commit set

The browser walkthrough surfaced a second pass, committed locally as well:

- The pill became strictly the fresh conversation's guide — it shows only on an
  empty transcript, and goes away the moment the pane is placed (a project
  picked, 不使用项目 confirmed) or a turn has been written.
- The confirm gate for content-bearing panes retired with the guide: a switch
  now only happens from a conversation that has nothing to lose.
- The palette opens quiet — no focused search box — and the search box matches
  the option rows' width.
- The sidebar's two hairlines agree: the 项目 header's line lost its private
  10px inset and now matches the 设置 footer's 4px.
- Task conversations file themselves by date — `…/2026/11/14/<name>.jsonl`
  under the auto workspace's own folder, the way the reference reads back as a
  calendar. `NewDatedSessionPath` is the task area's only; project sessions
  stay flat.
- The listing, the branch list, auto-archive and the empty-session GC all walk
  the dated subfolders (`eachSessionDir`), so nothing that scans a session
  folder loses a row.
- The window bar names a task conversation by its title alone — no `auto`
  folder crumb on a conversation that belongs to no project.
- The 任务列表 header lost its redundant + (new sessions start from 新建会话;
  projects are added from the pill and the empty state), and the scope menu
  now opens at the click, expanding toward the bottom-right, sized to its
  longest row.
- `reasonix serve --workspace-switch` and the two new refusal codes above are
  part of the same round.

### The FuchaZ round — thirteen commits transplanted by hand

The fork FuchaZ/DeepSeek-Reasonix:studio carries twenty-two commits of its own on
top of upstream and our cherry-picks. Thirteen were transplanted and adapted to this
tree — cherry-pick would have fought the pill, the task groups and the settings
rework, so each diff was applied and its conflicts resolved by hand:

- The right console's launcher and the memory body drop their width caps; the dock
  and nav slides ease gently; a live thought uses the whole window and the thought
  text carries its own weight token (`--w-think`); the narrow rail's deck chip
  yields; Patina's overlay gained its own step.
- The 扩展范围 pill portals its project list to the body — the settings scroll box
  clipped it; the scope menu opens at the click toward the bottom-right, sized to
  its longest row, its arrow riding the 最近 label (the count column is numbers
  only); the write-lease search result moved to the session page that hosts it.
- Kernel: the first turn asks the title model while it runs and the sidebar reads a
  real name (`titleauto.go`, gated on `agent.title_model` — the review round set it
  to the chat model); the meter rail's 对话轮次 counts authored turns, re-derived
  from the transcript; a loopback listener carries the provider grant.
- Skipped on purpose: the settings placement refactor (`be9120fb`, fights the
  会话互聊 rework — its own pass later), the notes-draft scripts, and the cherry-
  picked entries that were already ours.

### Open for the next session

- These commits are local only — push through the Git Data API pipeline
  (`C:\Users\Oliver\AppData\Local\Temp\reasonix-api-push\`), rerun CI, then delete and
  republish studio-v2.62.0 on the new head with the installer rebuilt.
- The token that once appeared in chat should be rotated; it is stored in the Windows
  credential manager and `~/.zcode/cli/config.json` only.

## 8. The rename fold, and 2.62.0 / 2.63.0 as two releases

The repo is renamed Oliver-Rui/DeepSeek-Reasonix-torch and its history is folded: the
rename commits and intro drafts ride the final intro commit (`docs(readme): the fork is
DeepSeek-Reasonix-torch…`), and the chain replays from anchor `55bd05f` through the Git
Data API so the pushed SHAs stay identical.

The release notes follow the two-release shape: `release-notes/studio/2.62.0.md`
is back with its refs remapped to the rewritten SHAs, and `2.63.0.md` tells only its
own release — the session-lands-first flow, dated task archive, first-turn naming,
the loopback grant, the FuchaZ trims, the QQ line and the rename.

Releases, in order: `studio-v2.62.0` tags the commit that completes the 2.62.0 set
and is built by dispatching the current workflow onto that tag (its tree predates the
publish gate); `studio-v2.63.0` tags the notes commit and builds from the tag push.
Both publish with the CI-built installer attached.

## 9. The 2.63.1 round: dated deletes, the queue ceiling, and the issue builder

The first install of 2.63.0 surfaced a family of breaks, all fixed on the same
day and shipped as `studio-v2.63.1` (tag on the tip, notes in
`release-notes/studio/2.63.1.md`):

- Task conversations refused delete, archive, export and batch removal with
  「该路径不属于任何已知的工作区」: the ownership check compared the session's
  parent to the sessions folder exactly, and the dated directories sat deeper.
- The boundary is the sessions tree now, dated directories included
  (`2cf2179`), proven by two serve tests and a live smoke against a real
  kernel (remember, archive, remove-many, forget — all 204).
- The queue strip lost its ceiling when the composer stopped routing it
  through `.composeaux`: a full inbox pushed the transcript to 0%.
- The row list's cap is back and folds with the window, and the cut-preview
  overlay (`.ovf`, hover half a second, portal to the body) returned with it
  (`49b28bf`). `perf/budget.mjs` guards this; a manual guard, not in CI.
- The corner notices rose above the plan card (z 5 → 8 against its 7), and the
  plan card meets the composer's edges (`inset: auto 0 100%`) (`49b28bf`).
- The batch bar ends with an X, and the blank-project picker starts at the
  platform's own last-used folder — `pickFolder` takes a preferred folder now,
  and "" hands the choice to the OS (`01fc659`).

Issue #2 closed with the family mark (`f44fd16`, Fixes #2): the model menu's
rows ask `markForModelId` — one keyword table in `vendormarks.tsx` — so a
relay's `deepseek-chat` shows DeepSeek and `glm-4.6` shows Zhipu.

An id nobody names keeps the door's mark or the letter chip, and a family
without artwork falls back until its svg lands in `assets/vendor-marks/`.

The release line learned two lessons, both now baked into the pipeline:

- Assets must never be uploaded from this machine: the egress proxy mangles
  multipart bodies and GitHub stores the raw request — 2.62.0's first
  installer gained a form-data prologue and Windows refused to run it.
  `attach-studio-asset.yml` attaches the installer from a runner instead,
  reading it back byte-identical.
- The tag is the version: `release-studio.yml` resolves `studio-vX.Y.Z` from
  the ref alone, so a patch release is notes plus tag, no source bump.

Four CI rounds chased timing flakes to green: the pill test awaits its
departure, the render-count windows retry, the footer notice query stays
inside the foot where the one notice lives.

The issue bots are wired but waiting on a key. `issue-builder.yml` turns a
`@builder` comment (or an `agent:build` label) into a pull request through
Z.ai's Anthropic endpoint, gated on `ZAI_API_KEY`; auto-labelling reads the
same route.

No public network exists here and Copilot's coding agent is paid-only, so the
key is the path. Until it lands, both jobs skip with a notice — and the repo's
own model family builds its own features the day the key arrives.

## 10. The 2.64.0 round — timeline sidebar, drag everywhere, message times

The round shipped as seven local commits (`14f8c4d`…`e4f77d3`, unpushed); the
session that followed added §11 on top, also uncommitted. What has run:
`tsc --noEmit`, census + census:verify (exit 0), focused suites, the WebUI
preview builds. What §11 added and what still waits is at the end of §11.

### Sidebar as a timeline (`desktop/frontend-next`)

- **Empty states**: 项目 / 任务 are standing controls (`WorkspaceGroupHead`),
  each with its own fold chevron and gray 暂无项目 / 暂无任务 notes that fold
  with their run.
- **任务列表 folds both runs at once** (`workspace.fold-all` `data-value="both"`,
  keys `session.collapse/expand_all_groups`).
- **Group drag**: dragging one label onto the other swaps the two runs; the
  choice rests in `state/railGroupOrder.ts` (`localStorage
  "rx-rail-group-order"`). MIME kind `application/x-reasonix-group`; visuals
  on `.rail .wsgroupline[data-drop]` / `[data-dragging]` in `design/sidebar.css`.
- **最近 moved out of the scope menu** into its own clock button left of 全部
  (`SessionFilter.tsx`, `state/railTimeGroups.ts`). Scope rows ink when
  chosen — no ✔, no 清除筛选 row.
- **Flat timeline reading** (clock on): every conversation is a two-line
  row (`.ds-row[data-sub]`), newest first, gathered under day buckets
  (`ui/timeBuckets.ts` — `dayBucket` walks the render order). Right side
  shows relative dates.
- **SessionMenu trim**: no session-info head, no 上移/下移 (`onMove` / `canMove`
  props gone, `onMoveTo` kept). Keyboard Alt+↑/↓ reorder stays
  (`Workspaces.tsx` row keydown → `useSessionMove.move`).
- **Session-row drag**: folder-tree rows only. MIME kind
  `application/x-reasonix-session`, apart from the group kind. A drop names a
  destination row and edge; `useSessionMove.moveTo` walks `hub.moveSession`,
  reloading **once**. Targets: same project, same archived flag.

### Composer and meter

- 对话轮次 became a button with a repeat icon beside the cache chip
  (`MeterRail.tsx`, `.studio-turns-anchor` / `.studio-turns-detail`), opening
  an info card like the cache one (keys `composer.turn_count_about/explain`,
  `turns_count_n`, `no_turns_data_yet`).
- The composer's send / pause buttons shrank to 75% (24px, icons 13.5px) and
  centre on the text row's Y (`design/composer.css`).

### Vendor marks and the magpie adaptation

- Ten SVGs copied from yetone/magpie into `assets/vendor-marks/`
  (magpie, xai, copilot, cursor, modelscope, groq, antigravity, baiducloud,
  huaweicloud, tencentcloud). `vendormarks.tsx` gains `PROVIDER_KEYS` +
  `markForProvider(name)`; a door's mark is its own label, not the family its
  models belong to.
- Kernel: `fetch_models.go` parses `display_name` into
  `ListedModel.DisplayName`; `probe.go` carries the names through the probe
  (`Probe.Names`); `POST /providers/refresh-models` merges the fetched list
  into the entry and prefills blank aliases from the custom names.
- Frontend: `refreshModels()` through the ports, a 刷新模型 button in
  `ProviderDetail.tsx`, and `EditConn.tsx`'s refetch prefills blank aliases
  from the refreshed names.
- **隐藏共同前缀 retired**: the EditConn field, its state and its payload
  line are gone, with their i18n keys. The `hideModelPrefix` field stays on
  both `ProviderEntry` and `ProviderEdit` with compat comments — stored
  configs still parse, nothing acts on it.

### Message timestamps (chat)

- `say` items carry `at?: number` — when the answer settled. Live paths
  stamp `Date.now()` in `sealSay` and both `foldMessage` settles
  (`state/say.ts`); the history rebuild derives `closedAt` for the closing
  card only, so an untimed card shows no stamp rather than an invented one.
- Both cards reveal the stamp on hover: `SayCard` puts
  `<time class="say-time">` at the end of its `.acts` row, `UserCard`'s
  existing `user-time` is gated the same way; hover-reveal CSS sits next to
  the `user-time` rule in `styles/studio.css` (`.call > .c:hover` lifts it).
- The switch lives in 设置 → 对话显示, first card, as 「消息时间」
  (`ui/MessageTimesSetting.tsx` + `ChatDisplaySettings.tsx`), backed by
  `state/messagetimes.ts` (`localStorage "rx-message-times"`, default on).
  Keys `settings.message_times` / `settings.message_times_hint`.

### Verified / not verified

| Check | Result |
| --- | --- |
| `tsc --noEmit` (whole tree, 2026-11-15) | clean |
| `vitest run src/ui/workspace-group-heads.interaction.test.tsx` | 18/18 |
| kernel `provider_refresh_test.go` + `fetch_models_test.go` (earlier) | green |
| Full `vitest`, `vite build`, census, census:verify, repolint | **not yet run** |
| WebUI preview rebuild (ports 8787 / 8788) | **not yet done** — build, then serve with `--auth none --no-open --workspace-switch --page dist`; the per-restart token goes in via `POST /auth/token` from the page (a `#token=` fragment exchanges only on a full reload) |

### What the next session owes

1. `npm run build`; reload both local previews and let the user review —
   timestamps on hover, row drag, label drag, the timeline reading, the
   retired prefix toggle.
2. Full `vitest` + `npm run census` + `census:verify` + `repolint -changed`.
3. Batch commits, `release-notes/studio/2.64.0.md`, tag `studio-v2.64.0`,
   push through the Git Data API pipeline, then attach the installer **from a
   runner** — never from this machine.
4. Cosmetic: the forged 更早 demo session inherited the title 123 — rename it
   in the demo dir or leave it; it is review data, not product data.


## 11. The review round that finished 2.64.0 — drag lines, folder folds, magpie names

The user's UI review of §10 produced this round, uncommitted on `studio` on
top of the seven local commits. Everything below was built into the WebUI
preview (port 8787) and shown to the user as it landed.

### Drag, unified

- Every drag in the tree speaks one insertion line: a 2px accent bar on the
  edge the drop will land at, in `design/sidebar.css` — group-label rows,
  project rows and conversation rows all read it the same way.
- Session drops name a row **and an edge**; `useSessionMove.moveTo` seats the
  dragged row above or below the target (`before` from<to → `to-1`, else `to`;
  `after` mirrors it).
- **Group swap fixed**: the old semantics made the carried run the leader, so
  dragging 项目 onto 任务 while it already led was a silent no-op. The drop
  now flips `rx-rail-group-order`, and the line rides the target head's
  bottom edge.
- **Drag ghost**: `ui/dragghost.ts` sets a small name chip as the drag image —
  the browser's default snapshot of a whole row was the slow part of a drag.

### Sidebar regression fixes

- Run folds are the 2.64.0 rows-gone design (only the label stays), persisted
  in `state/railRunFold.ts` (`rx-rail-run-fold`) — the round had left them
  in-memory, and a search reveals folded runs' rows.
- The 2.64.0 Sidebar edit had **dropped `onFoldGroup` from the Workspaces
  mount**, so group heads' folds did nothing in the real app; restored.
- Per-project fold chevrons are gone: the folder icon itself reads closed /
  open (`folder` / `reveal`), and the row's left padding puts the icon on the
  nav rows' icon line.

### Group heads carry a plus

- 项目 and 任务 heads each gained a right-edge plus. The projects one opens
  a two-item menu at the button's bottom-left; the blank dialog moved to
  `ui/BlankProjectDialog.tsx`, shared with the pill. The tasks one opens a
  non-project conversation.
- The menu anchor is the button's rect **divided by the page's `--zoom`** —
  fixed positioning inside the zoomed root was the reason the first version
  floated away. The hover card shares the fix.

### Peers, surfaced

- The standing 会话互聊 bar above the composer is gone. Dispatched sessions
  read as one quiet chip in the meter rail (`DispatchThreadsBar` rewritten),
  and hovering it opens the small card with each peer's state and lamp.
- An unknown peer — the audit trail carries keys, not titles — reads
  「已派发的会话」（未知项目） instead of the word undefined.
- Root fix: `NoteDispatchedSession` now carries the role, and `/peers` joins
  roles from the sender's own memory (`DispatchRoles`), so a closed partner's
  row keeps its name. The hub's live-runtime walk is gone.

### The sidebar's hover card

- Hovering a session row (450ms) opens `ui/SessionHoverCard.tsx`: title,
  project, folder, last move, and one line per dispatched task, each opening
  the session it landed on. The kernel serves `GET /dispatch-threads?path=`
  from `dispatchBook.threads`, recorded by sender session path.

### Model names come from the door

- `ModelShort` draws each model by its last path segment, by default — a
  relay's `group/…`, `auto/…`, `qoder-cn/…` prefixes leave the menus. Two
  doors onto one model read alike on purpose; the door heading distinguishes.
- The startup pass (`autoRefreshDisplayNames`, once per process) reads every
  door's listing and prefills blank aliases from its `display_name` — with
  no hand.
- `vendormarks.tsx` keeps a mark table for ids that name no family (qoder's
  `gfmodel` → zhipu, `fast` → qwen, …); the names themselves are never
  guessed. Five preset vendors gained marks, gathered from Simple Icons and
  the vendors' own sites.

### Checks and debts

- Focused suites green: the sidebar/fold/drag cluster, marks and menus,
  peers, `actions.test` with the ceiling carried at its honest 215. Census
  and census:verify exit 0. Kernel peer, dispatch, provider and model tests
  green; five machine-only config failures predate the round (CI verifies).
- repolint: `Workspaces.tsx` (1086 lines) and the css/file-size totals are a
  deliberate carry-forward of the round's growth — baseline updated below;
  splitting `Workspaces.tsx` is the follow-up that earns the ceiling back.
- What still waits: the batch commits, `release-notes/studio/2.64.0.md`, the
  tag, the Git Data API push, the runner-attached installer — the user
  reviews first, and pushes only on their word.

### Commit batch plan (files that share no other batch)

1. kernel models: `internal/…/{providers.go,probe.go,fetch_models.go,fetch_models_test.go}`
2. providers UI + marks + retired toggle: `vendor-marks/*.svg`, `vendormarks.tsx`,
   `ServiceMark.tsx`, `port/{provider,port,sse_provider,mock_provider}.ts`,
   `ui/{Providers,ProviderDetail,EditConn}.tsx`, `msg/models.ts`,
   `en_settings.ts`, `ModelPrefix.test.tsx`
3. sidebar round: `ui/{Workspaces,SessionMenu,SessionFilter,Sidebar,ProjectsHeader,WorkspaceGroupHeaders,useSessionMove}.tsx*`,
   `ui/timeBuckets.test.ts`, `state/{railGroupOrder,railTimeGroups}.ts`,
   `design/{sidebar,rows}.css`, `msg/session.ts`, `en_window.ts`, the six
   sidebar interaction-test files
4. composer/meter: `design/composer.css`, `ui/MeterRail.tsx`, `msg/composer.ts`
5. timestamps: `state/{session_types,say,session,messagetimes}.ts`,
   `ui/cards/{SayCard,UserCard}.tsx`, `ui/{ChatDisplaySettings,MessageTimesSetting}.tsx`,
   `msg/settings.ts`, `styles/studio.css`
6. registry: `src/actions.ts` (`session.sort`, `provider.refresh`)
7. this document

### The CI round that followed the push (2026-11-15 night)

The user said push; the 21-commit delta went up byte-identical, the tag was
cut, the release published, and the runner attached the installer — and the
push's own CI caught four things the machine had not run since the batch
rework. Each fixed at its root:

- `select.test` parsed a sheet's comments out once and then re-execed the
  raw text for every following rule, so a comment's own words (" select" in
  the picking bar's prose) surfaced as a selector. The whole walk now reads
  the stripped text.
- The picking bar's rework left six `workspace.*` keys in the session table
  and three removed `session.remove_*` keys behind. The six now spell
  `session.*`; the three are gone.
- The rename dialog's close button and its input's onChange were unnamed
  inputs — the two that pushed the census to 217 over its 215 ceiling. The
  button is `workspace.rename-close`, the input declares
  `data-action-change="workspace.rename"`, and the ceiling holds at 215.
- The worktree test still asked for a 从列表移除 menu item; it reads the
  menu's own 移除 now.
- Kernel: `probe_test.go` followed `describe`'s `names` argument (nil at all
  three call sites), `provider_model_short_test` asserts the fixed
  last-segment short instead of the retired gate, and `/dispatch-threads`
  joined vite's ROUTES.
- `perf/session-move.mjs` still right-clicked for 上移/下移 menu items the
  slimmed menu no longer renders; it moves with Alt+Arrow on the focused row.
- The Range.getClientRects stub moved into `testkit.ts` (the shared floor),
  retiring the four per-file copies; the occlusion hit-test stops logging
  unhandled TypeErrors in every tooltip-adjacent test.
- repolint baseline widened for the last three commits' growth (actions +5,
  Workspaces 286→333, hub_tree +63, the 2.64.0 notes) — same deliberate
  carry-forward as above.

### The second CI round (2026-11-15 late night) — three subagents, one coordinator

The re-cut tag's CI still held four reds; three parallel subagents carried
them down while the coordinator took the one their concurrency limit
returned:

- `test (ubuntu/macos)` never reached a test — both die at the gofmt step:
  - The round's kernel files were written without a format pass, and only
    the Unix legs run gofmt (the Windows checkout's CRLF would flag
    everything). Six files formatted; Windows skipping the step is why the
    machine never saw it.
  - The coordinator's GOOS=linux/darwin build+vet (via goproxy.cn —
    proxy.golang.org is walled here) proved nothing else hides in the
    unix-only surfaces.
- `lint` held one dead method: `dispatchBook.rolesFor` lost its reader when
  /peers moved to the sender's own DispatchRoles memory. A subagent
  verified the zero callers, removed it, and proved four serve-package
  failures unrelated via a pre-removal worktree.
- `browser guards` ran its first honest lap in two rounds (the session-move
  guard used to stop it early):
  - control-icons found the send circle's 13.5px icon carrying lucide's
    14-unit glyph at 7.875px — the app's icon system is 16px, and the
    guard's ≥9 threshold was written for it. Back to 16px, fits and
    centering intact.
  - Behind it sat a second stale guard: project-header still walked the
    filter menu for the retired 最近 row. It asserts the menu carries no
    ordering row, and exercises the header's timeline clock (aria-pressed
    flips, `.wstime` sections come and go) on the long list instead.
  - 4 further guard failures were load flakes of the local parallel run —
    each passes alone and on CI.

- The Studio run's own frontend-next job then flaked once: the update-notice
  test (the render-count-window one, retry×2 already) timed out 92s three
  attempts on a 12-job parallel burst, while the same tree passed the full
  suite on this machine twice. Rerun of the failed job: green.
  - A follow-up for its designed round: make the notice's window time- or
    publish-event-based instead of render-count-based, and the flake dies
    with it.

## 12. 2.64.0 out the door — the shipped state (2026-11-16)

This section supersedes §10's "Verified / not verified" and "What the next
session owes" tables: they were the mid-round truth, and everything they
list as owing has since been done.

**Shipped.** Tag `studio-v2.64.0` sits at `4dba223`; the branch tip is
`ab62ab6`, docs-only above the tag. The release carries the bilingual notes
and the Windows installer attached from a GitHub runner (this machine's
egress proxy corrupts multipart uploads — never upload binaries from here):
<https://github.com/Oliver-Rui/DeepSeek-Reasonix-torch/releases/tag/studio-v2.64.0>.

On the tagged tree every workflow is green:

- Release Studio, Attach studio asset, Deploy Web Studio
- CI: test/race/coverage/vet/gofmt/lint across the matrix
- Studio: frontend-next 6883 tests, browser guards

Local previews on :8787 (real data) and :8788 (empty demo) run the final
build.

**The git record, in one breath.** e4f77d3 closed the seven-commit 2.64.0
round; fourteen review commits followed (drag lines, folder folds, plus
menus, peers chip and hover card, magpie door names, batch bar, project
menu).

Then the push's own CI caught what the machine had stopped running: five
commits of root-cause fixes (81bcce5..a048e7d), and its re-run caught four
more (d731dff..9a73624, via three parallel subagents); docs carry the rest.

Thirty-one commits cross the 2.63.1..2.64.0 span, every one byte-identical
on the remote — local SHAs are valid citations.

**Debts and follow-ups, consolidated:**

- The update-notice test's render-count window flaked once under a 12-job
  runner burst; make the window time- or publish-event-based in its designed
  round.
- `Workspaces.tsx` (~1100 lines) is a deliberate baseline carry; splitting
  it is the extraction that earns the ceiling back.
- The MiMo roster gaps: busy/purpose fields on session workers, sync-wait
  semantics for talk, an external-agent roster, a purpose tool
  (update_session's equivalent).
- 更改文件夹位置 stays deferred — session history is keyed by path hash, so
  a move orphans sessions; it needs its own designed round.
- This machine alone fails 5 config scoping tests, part of
  `internal/session/control`, `workspacelease`, and builtin's
  `TestGrepSkipsVendorDirs`.
  - No runner reproduces any of them — machine noise, do not chase.
- The GitHub token seen in chat on 2026-11-14 is still unrotated.

**Publishing recipe** (scripts in `%TEMP%\reasonix-api-push`, token via
`git credential fill`):

- push_robust.py with `WALK_BASE` = the remote tip's full sha, feat
  fast-forwarded to studio.
- To move a tag, delete `git/refs/tags/<tag>` first and let release_2.64.py
  recreate it — the release POST 422s and reuses, the attach run
  re-dispatches.
- TIP_SHA always comes from `git rev-parse studio`, never from memory. All
  the pipeline's scars are in the pitfalls notes.

## 13. The 2.64.1 round — the drags were never wired to the kernel they serve

Four user reports, one patch release. What each turned out to be:

- **Sessions could not be reordered, projects either.**
  - A kernel-level smoke (curl with the launch-token cookie) showed
    `/tree/sessions/move` answering 403 `session.outside_workspace` for the
    very row the tree lists first.
  - The two sides enumerated differently: the tree reads the project page —
    where a pane's in-flight first turn lives only in lease and inbox
    sidecars — while the move validated against a raw store walk.
  - The move now validates and orders against the tree's own rows —
    `treeSessionRows` feeds move, pin, viewed and move-to; in-flight rows
    answer 204 and stay pinned above, dispatch transcripts refuse, archived
    partitions step inside themselves.
  - A real browser drag (Playwright dispatching the HTML5 events) then
    reordered projects and archived sessions with 204s and persistence across
    reloads. Unremembered workspace rows are not draggable — they are not
    members of the remembered order — which the smoke learned the hard way.
- **The uninvited "Oliver" row** was the launch root resolved to the home
  directory twice over.
  - The host inherits the shell's working directory and the picker counted
    every non-install directory as chosen; a stray `.git` in the home
    directory made the boot's nearest-git-root walk crown it a project.
  - The launch refuses the home directory now; removal writes a tombstone
    into the workspace list (bounded at 64, never for the auto workspace) so
    an implicit root stays gone until the folder is added back.
  - The startup splash is a 3-second handshake threshold, not a dedicated
    wait; with the launch root off home, the per-root boot work stops hitting
    the home tree.
- **The version panel read a mirror this fork does not have.** The catalog is
  the GitHub releases list now, and the mode routes answer without a declared
  install so the background watch runs.
  - Rows run newest first with the running build marked; `newer` lights the
    railfoot; each row carries its installer as a plain download link; notes
    read from the release body.
  - A dead GitHub answers with err beside the data, so the panel never loses
    which version is running.
- **The crew docs**: dispatch's role now names crew members (组员A：美术 /
  组员B：开发), send_peer_message is the worker's report channel, and both
  guides describe the router-and-named-crew pattern.

Subagents carried the kernel move fix and the phantom-workspace hunt (their
one shared-file hazard: a `git reset` mid-flight — index-only, the working
tree survived; the rule after: never touch git state from a subagent). The
release pipeline moved to tag `studio-v2.64.1`.

### The drag that painted a line and moved nothing (same night)

A real-input probe — page.mouse starting the native HTML5 drag, not a
synthetic dragstart — found the gap the synthetic smoke had hidden: both
"natural" gestures were silent no-ops.

- Dropping a row on the **next row's top edge** asks for "before it", which is
  exactly where the row already sits; the line painted, the drop returned
  zero steps. The complementary downward edge worked, and far drops worked,
  which is why every earlier test passed: they dragged non-adjacent rows.
- Fix: on the immediate neighbour the edge **flips** — carried down it lands
  under that row, carried up it lands over it — and the painted line follows
  the flipped edge, so a line always means a move.
  - Sessions and projects use the same rule (`Workspaces.tsx` dragover,
    `useWorkspaceDrag.ts`); the tests
  that had encoded the old no-op now assert the flip, plus a regression test
  that a reorder writes no tombstone (`TestReorderingProjectsWritesNoRemoval`).
- The user's three projects had disappeared from their sidebar with
  tombstones recorded: a removal is explicit and confirmed, but nothing about
  a *reorder* should ever write one — the new kernel test locks that. The
  three folders were added back through the add route, which lifts tombstones.
