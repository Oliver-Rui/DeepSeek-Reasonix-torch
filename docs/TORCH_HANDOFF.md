---
owner: @SivanCola
backup: @esengine
status: active
reviewed: 2026-11-24
---

# main handoff — 2.67.2 published (Windows only)

The state of the fork after the 2.67.2 release: what shipped, how it was verified,
and what is still open.

The line was named `torch` through 2.67.0 and returned to `main` on 2026-11-21. The
fork's first revision is described in `docs/STUDIO_TWEAKS_HANDOFF.md`; this document
supersedes it.

## 1. Where things stand

| Item | State |
| --- | --- |
| Release | [`studio-v2.67.2`](https://github.com/Oliver-Rui/DeepSeek-Reasonix-torch/releases/tag/studio-v2.67.2) published 2026-11-24 23:13 by a `platforms=windows` dispatch, and it is GitHub latest. 15 assets: the four Windows packages, a `.minisig` beside each, and the six CLI archives with `SHA256SUMS`. No macOS or Linux Studio package is in it, and the release's standing paragraph says so in as many words. The packages are minisign-signed but not Authenticode-signed (`STUDIO_SIGNING_ENABLED` stays unset), so SmartScreen still warns on first install. 2.67.1 remains at [`studio-v2.67.1`](https://github.com/Oliver-Rui/DeepSeek-Reasonix-torch/releases/tag/studio-v2.67.1) (35 assets, six platforms); 2.67.0 at [`studio-v2.67.0`](https://github.com/Oliver-Rui/DeepSeek-Reasonix-torch/releases/tag/studio-v2.67.0) (21 assets, unsigned) |
| Branches | `main` only (renamed from `torch` on 2026-11-21), and it is the default branch. `studio` and the nine `area/*` branches are deleted — each verified contained first |
| History | One continuous line from `baseline: upstream 2.61.0` through the 2.66.1 fork work and every change since; the pre-rewrite full history (13,451 commits, upstream + fork through 2.66.1) lives in the `archive/pre-reset-2.66.1` tag on GitHub, alongside `upstream-v2.61.0` |
| CI | green on `540014a1c` and `ab5127ffa`; from 2026-11-24 the macOS runners were cancelled about fifteen minutes after queueing, which failed CI, Studio and the first 2.67.2 release run on their macOS jobs while every other leg passed. The Windows-only release path in section 7 exists because of it |
| Tag | `studio-v2.67.1` at `540014a1c`, the commit its assets were built from. `studio-v2.67.2` is on the remote at `0917fd6ea`, the main tip its assets were built from: it was first placed at `d2b741752` and then moved, because the publish job checks out the tag's commit and a release-path script added afterwards would not be there to run. No application code differs between the two commits |
| This machine | Sandbox on (`[sandbox] windows_integrity = true`), so Git Bash exits at start under the write-restricted token and the shell tool runs `pwsh`; `[tools.shell] prefer = "pwsh"` names that interpreter and stops the per-boot warning — commands here are PowerShell. pnpm 12 is installed globally for local packaging, and `dist/ReasonixStudio-windows-amd64-installer.exe` (2.67.2, unsigned) is the untracked local build |
| Working tree | Two deliberate local changes, not committed: `xiaomimimo.svg` (vendor mark touched) and `dist-local2/` (a local electron build) |

## 2. What 2.67.0 ships

- **Session chat and cross-project dispatch are removed** (`ec0fe1e88`). The peers spool, audit,
  HMAC proofs, dispatch board and every `peers.*` setting are gone; sessions run subagents in
  place (`internal/runtime/delegation` untouched).
- **Removing a project keeps its archive reachable.** The removed project appears as a dimmed
  ghost row in the Archived scope and in search results; deleting the last conversation retires
  the row and its record together; the row's menu offers 放回列表.

## 2.5 What 2.67.1 ships

- **The narrow fold keeps every control inside the pane.** Three roots, and the fixes
  that answered each:
  - Viewport clamps divided by nothing: the twelve clamps across six sheets used bare
    100vw while the shell lays out at 100vw / --zoom, so panels escaped a zoomed narrow
    window by exactly the zoom factor. Fixed with the division everywhere plus a contract
    test banning bare viewport subtractions.
  - Layer order outranked specificity: the design layer's `flex: none` on the model group
    and control beat every legacy squeeze rule, so the pill overflowed its pane by 40px
    at 320px until the shrink chain moved into the design layer itself, with the name
    truncating across the whole squeezed range in `phone.css`.
  - The collapsed pill kept two 14px icons in a 30px box. Fixed with the chevron hidden
    like its sibling collapses and the pill row paired to the card's exact fold width;
    tips are clamped by cqw, the drawer is sized against the zoomed viewport, and the
    model menu is capped.
- **The web access panel speaks the settings row language.** It shipped with no stylesheet
  at all — native bullets, the reorder buttons and the switch each on their own line. The
  rows adopt `lrow` (the class names already matched) and `webaccess.css` fills in the
  panel's own pieces.
- The repo's only branch returned to `main` (CI, CodeQL, Studio, the release workflow, the
  signing contract and the docs' branch gates all follow the rename).

## 3. The release line follows main

`release-studio.yml` now validates that a `studio-v*` tag sits on `main` history; CI, CodeQL,
Studio, acp-conformance and the certum smoke run on main; the signing contract allows the main
branch; `docs/STUDIO_RELEASE.md` fetches main. The tag namespace is unchanged.

## 4. How 2.67.0 was verified

Local gates: `go build`, `go vet`, `repolint` clean, frontend `tsc` + `vitest` (919 files),
Go full suite. CI went green only after nine real repairs it surfaced:

- lint orphans, and a stale wire-parity table naming deleted peer files
- the signing contract's branch, and a legacy-tombstone test on an 8.3 temp path
- the settings guard's 390px overflow, and a stale hint assertion
- the price table's 90-day gate, and two browser guards waiting on deleted selectors
- one unnamed mutating input in the census

Two packages stay red **on this machine only**: `cmd/e2ebench` and `cmd/signpath-contract`.
The Windows Store `python3` stub exits 49 and `jq` is not installed; both pass on CI Linux.

The drag flow was exercised end to end in the running app: drop → reference card ready → spacing
measured on three viewport heights → card retired on delete.

### How 2.67.1 was verified

Frontend `tsc` clean and `vitest` 42/42. Three contract files pin the fold's structure:
`narrow-fold.test.ts` pins the 30px collapse, the squeeze-range truncation, the
design-layer shrink chain, the drawer and menu caps, and the pill-row pairing.

`zoom-safe-viewport.test.ts` requires every viewport subtraction to divide by --zoom,
and the web-access interaction suite pins the row structure.

The fixes were then measured on the production build in the running app: a DOM sweep at
320/360/480/720 found zero viewport escapes, the pill row aligned to the card at Δ2–4px,
and the model menu computed to its viewport cap.

The lesson worth keeping: specificity does not cross layer order. The design layer's
`flex: none` beat a (0,3,0) legacy rule; only a rule inside the design layer wins there.

### How 2.67.2 was verified

The release path itself was checked before it was trusted: the matrix script's five
cases, the artifact verifier's seven behaviours (all platforms, the Windows set, a
missing architecture, an unnamed set, an unknown platform, an empty file), and the
resolve step end to end.

With `platforms=windows`, `GITHUB_OUTPUT` holds two legs and nothing else, and the
standing paragraph was run for both platform sets and read.

In the run, `resolve`, the signing contract, both Windows build legs, both install
smokes, the gate and `publish` all succeeded. The manifest carries exactly
`windows-amd64` and `windows-arm64`, and the release holds 15 assets with a `.minisig`
beside each Windows package.

## 5. Open items

- **The Studio catalog still names 2.67.0.** No `R2_*` secrets are configured, so the
  publish job skips the mirror and the catalog write, and a running install is offered
  neither 2.67.1 nor 2.67.2 — the 2.67.2 run logged exactly that warning. Configure the
  four R2 secrets to resume mirror and in-app updates.
- **Installs older than 2.67.1 cannot self-update.** The signing key rotated, so an
  install at 2.67.0 or earlier fails verification against the key embedded in 2.67.1
  and has to download the package from the release page by hand.
- **Windows packages are minisign-signed, not Authenticode-signed.**
  `STUDIO_SIGNING_ENABLED` stays unset, so SmartScreen warns on first install; the
  release notes say so and name the click-through.
- `ROADMAP.md` R5 blocks the next **stable** `studio-v2.*` tag until the GA gates (G1, G3, G4,
  G5) are met. 2.67.0, 2.67.1 and 2.67.2 were tagged at the user's explicit request; a fork release
  moves no npm pointer, but the gate note stands for anyone reading the runbook.
- `checkedOn` in `internal/contract/pricing/catalog.go` is 2026-11-20; the 90-day gate goes red
  again around 2027-02.
- **A release whose tag predates a release-path change cannot be dispatched.** The first
  2.67.2 dispatch failed in `Collect artifacts`: the publish job checks out the tag's
  commit, so the verifier it called there was the one from before the change.
- **Move the tag instead.** A tag recovered by a newer workflow has to sit at or after
  the commit that carries the release path, which is why 2.67.2 now points at the main
  tip rather than the commit it was first placed on.
- 2.67.0's assets were published by hand from the run's `studio-build-*` artifacts, with
  curl resuming because the network to the artifact CDN resets under load.

## 6. What the publish round repaired

The tag was first pushed a commit ahead of the fixes, so its run failed the check that a
`studio-v*` tag sits on `main`. The tag then moved to the repaired commit and the workflow
was re-dispatched with `tag=studio-v2.67.1`, which rebuilt every platform and published
from that run.

Eight repairs came out of the round, and each carries a guard:

- **The embedded-key test** (`internal/platform/update/update_test.go`) asserts the key ID
  embedded in `verify.go`; a rotation that skips it fails every CI leg.
- **The publish job's condition** (`release-studio.yml`) spells the skipped-Certum branch
  out itself. A bare `needs.publish-gate.outputs.publish == 'true'` still applies an
  implicit `success()` over skipped needs and skips the publish the gate approved.
- **`cmd/sign`** (`desktop/cmd/sign/main.go`) reads an unencrypted key with
  `UnmarshalText` when `minisign.IsEncrypted` is false. The live key is stored
  unencrypted (KDF none, the comment still saying "encrypted"), so `DecryptKey` alone
  refused it; `MINISIGN_PASSWORD` stays unset and a test pins that shape.
- **The two style contracts**: `.peerfield` keeps only its width cap and type because the
  kit draws the box, and the narrow model name carries `line-height: 1.2` beside its
  ellipsis (`field.test.ts`, `linebox.test.ts`).
- **The budgets**: each layered sheet merged one line back to its recorded length and the
  handoff prose splits where a block passed 320 columns; `repolint -update` was not used.
- **`v1-maintenance-schedule.yml` is deleted**: it dispatched the 1.x monitors on
  `main-v2`, which no longer exists, so every scheduled run failed with a 404. Restore it
  with the branch and those monitors if the 1.x line returns.
- **The sandbox comment** (`internal/contract/config/render.go`) names `windows_integrity`
  as the switch that turns Windows bash on; the save/reload guard reads the assignment
  and reloads its own output, so a comment may name the key.
- **The vulnerability gate**: `govulncheck` began flagging GO-2026-6278 in
  `github.com/gorilla/websocket` v1.5.3 (weak PRNG for the client mask key), so the
  module moves to its fix commit. That commit carries no tag; the next upgrade should
  prefer a tagged release once one exists.
- **The permission menu** (`desktop/frontend-next/src/ui/Policy.tsx`) lists only the four
  modes: the 沙盒与运行边界 row and the note beneath it are gone, and `Policy.test.tsx`
  pins that the menu carries neither a settings entry nor a note.
- **The side columns were tried and withdrawn.** One change made both immune to window
  width — the rail's `foldaway` call and the workbench's 760px media rule — and the owner
  cancelled it before release, so `a52e0c4fb` and its notes commit are reverted in
  `538dd7ed9` and `d2b741752`; the width-driven fold stands.

The round's commits run from `1164342aa` (the key test) to `d2b741752` (the revert). The
2.67.2 notes sit in `release-notes/studio/2.67.2.md`.

## 7. The Windows-only release path

The macOS runners were being cancelled about fifteen minutes after queueing, and one
cancelled `build (darwin/amd64)` leg was enough to stop the gate from publishing
anything — the Windows packages included. So a release can now be dispatched with a
narrower platform set.

- `platforms` is a `workflow_dispatch` input — `all` by default, `windows` the other
  value — so pushing a tag still builds and publishes the whole line, and only an
  explicit dispatch narrows one.
- `resolve` decides the set once and hands `build` a matrix built by
  `scripts/studio-build-matrix.sh`, which holds the six legs with their runners,
  architectures and artifact names. A release that claims only Windows expands two
  legs; the platforms it does not claim have no leg at all.
- `publish` stages, verifies and describes only what the release claimed: the standing
  paragraph names the platforms the release carries instead of a `.dmg` nobody
  uploaded. The CLI job is untouched — it builds all six targets from one Linux
  runner, so a Windows-only Studio release still carries the full CLI set.
- A job-level `if` cannot read `matrix`: the first cut put the platform claim there
  and GitHub refused the workflow outright (`Unrecognized named-value: 'matrix'`).
  `studio.yml` already expands its packaging matrix from a job output, which is the
  pattern used here.

Two things cost a run each and neither was the change. A Windows runner's
electron-builder cache held a truncated 7zip (`ENOENT ... electron-builder\Cache\7zip@1.0.0`),
cleared by rerunning the failed job.

And creating or moving a tag ref through the API fires a `push` event, which starts a
full-line run in the same concurrency group; cancel it before dispatching.
