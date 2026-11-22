---
owner: @SivanCola
backup: @esengine
status: active
reviewed: 2026-11-22
---

# main handoff — 2.67.1 shipped

The state of the fork after the 2.67.1 release: what shipped, how it was verified, and
what is still open. The line was named `torch` through 2.67.0 and returned to `main`
on 2026-11-21. The fork's first revision is described in `docs/STUDIO_TWEAKS_HANDOFF.md`;
this document supersedes it.

## 1. Where things stand

| Item | State |
| --- | --- |
| Release | [`studio-v2.67.1`](https://github.com/Oliver-Rui/DeepSeek-Reasonix-torch/releases/tag/studio-v2.67.1) published 2026-11-22 from the release run's artifacts by the automated hand-publish pipeline (see §5); 2.67.0 remains at [`studio-v2.67.0`](https://github.com/Oliver-Rui/DeepSeek-Reasonix-torch/releases/tag/studio-v2.67.0) (21 assets, 2.49 GB, six platforms) |
| Branches | `main` only (renamed from `torch` on 2026-11-21), and it is the default branch. `studio` and the nine `area/*` branches are deleted — each verified contained first |
| History | One continuous line from `baseline: upstream 2.61.0` through the 2.66.1 fork work and every change since; the pre-rewrite full history (13,451 commits, upstream + fork through 2.66.1) lives in the `archive/pre-reset-2.66.1` tag on GitHub, alongside `upstream-v2.61.0` |
| CI | frontend `tsc` clean and `vitest` 42/42 on the release commit; the CI and Studio failures on the tagged commit were the comment-budget lint (fixed in the release commit itself) and a CI-only test flap (rerun green locally; confirm on the rerun) |
| Working tree | Two deliberate local changes, not committed: `xiaomimimo.svg` (vendor mark touched) and `dist-local2/` (a local electron build) |

## 2. What 2.67.0 ships

- **Session chat and cross-project dispatch are removed** (`ec0fe1e88`). The peers spool, audit,
  HMAC proofs, dispatch board and every `peers.*` setting are gone; sessions run subagents in
  place (`internal/runtime/delegation` untouched).
- **Removing a project keeps its archive reachable.** The removed project appears as a dimmed
  ghost row in the Archived scope and in search results; deleting the last conversation retires
  the row and its record together; the row's menu offers 放回列表.

## 2.5 What 2.67.1 ships

- **The narrow fold keeps every control inside the pane.** Three roots: twelve viewport
  clamps across six sheets divided by nothing (100vw is visual while the shell lays out at
  100vw / --zoom, so panels escaped a zoomed narrow window by exactly the zoom factor);
  the design layer's `flex: none` on the model group and control outranked every legacy
  squeeze rule by layer order regardless of specificity, so the model pill overflowed its
  pane by 40px at 320px until the shrink chain moved into the design layer itself; and the
  collapsed pill kept two 14px icons in a 30px box. Fixed with: the division everywhere +
  a contract test banning bare viewport subtractions, the name truncating across the whole
  squeezed range with the shrink chain in `phone.css`, the chevron hidden like its sibling
  collapses, the pill row paired to the card's exact fold width, tips clamped by cqw, the
  drawer sized against the zoomed viewport, and the model menu capped.
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

Frontend `tsc` clean and `vitest` 42/42 — three contract files pin the fold's structure:
`narrow-fold.test.ts` (the 30px collapse, the squeeze-range truncation, the design-layer
shrink chain, the drawer and menu caps, the pill-row pairing), `zoom-safe-viewport.test.ts`
(every viewport subtraction divides by --zoom), and the web-access interaction suite (the
row structure). The fixes were then measured on the production build in the running app:
a DOM sweep at 320/360/480/720 found zero viewport escapes, the pill row aligned to the
card at Δ2–4px, and the model menu computed to its viewport cap.

The lesson worth keeping: specificity does not cross layer order. The design layer's
`flex: none` beat a (0,3,0) legacy rule; only a rule inside the design layer wins there.

## 5. Open items

- The release is **unsigned**: `MINISIGN_PRIVATE_KEY` is not configured, so the workflow's
  publish job skips by design. 2.67.0's assets were published by hand from the run's
  `studio-build-*` artifacts; 2.67.1's by an automated hand-publish pipeline (curl with
  resume, because the network to the artifact CDN resets under load). Configure the key
  (and `STUDIO_SIGNING_ENABLED`) for CI to publish and sign itself.
- `ROADMAP.md` R5 blocks the next **stable** `studio-v2.*` tag until the GA gates (G1, G3, G4,
  G5) are met. 2.67.0 and 2.67.1 were tagged at the user's explicit request; a fork release
  moves no npm pointer, but the gate note stands for anyone reading the runbook.
- `checkedOn` in `internal/contract/pricing/catalog.go` is 2026-11-20; the 90-day gate goes red
  again around 2027-02.
- The 30-minute release-progress automation created for this handover was deleted on completion.
