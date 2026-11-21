---
owner: @SivanCola
backup: @esengine
status: active
reviewed: 2026-11-21
---

# torch handoff — 2.67.0 shipped

The state of the `torch` line after the 2.67.0 release: what shipped, how it was verified, and
what is still open. The fork's first revision is described in `docs/STUDIO_TWEAKS_HANDOFF.md`;
this document supersedes it.

## 1. Where things stand

| Item | State |
| --- | --- |
| Release | [`studio-v2.67.0`](https://github.com/Oliver-Rui/DeepSeek-Reasonix-torch/releases/tag/studio-v2.67.0) published 2026-11-20, 21 assets (2.49 GB), six platforms built and install-smoked |
| Branches | `torch` only, and it is the default branch. `studio` and the nine `area/*` branches are deleted — each verified contained in `torch` first |
| CI | 45/45 green on the release commit `11c59e4d4`; five UI commits landed after it |
| Working tree | Two deliberate local changes, not committed: `xiaomimimo.svg` (vendor mark touched) and `dist-local2/` (a local electron build) |

## 2. What 2.67.0 ships

- **Session chat and cross-project dispatch are removed** (`c2e58be4a`). The peers spool, audit,
  HMAC proofs, dispatch board and every `peers.*` setting are gone; sessions run subagents in
  place (`internal/runtime/delegation` untouched).
- **Removing a project keeps its archive reachable.** The removed project appears as a dimmed
  ghost row in the Archived scope and in search results; deleting the last conversation retires
  the row and its record together; the row's menu offers 放回列表.
- The presentation was chosen by the user from two candidates; the everyday list stays clean.

## 3. The release line follows torch

`release-studio.yml` now validates that a `studio-v*` tag sits on `torch` history; CI, CodeQL,
Studio, acp-conformance and the certum smoke run on torch; the signing contract allows the torch
branch; `docs/STUDIO_RELEASE.md` fetches torch. The tag namespace is unchanged.

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

## 5. Open items

- The release is **unsigned**: `MINISIGN_PRIVATE_KEY` is not configured, so the workflow's
  publish job skips by design and the artifacts were published by hand from the run's
  `studio-build-*` artifacts. Configure the key (and `STUDIO_SIGNING_ENABLED`) for CI to
  publish and sign itself.
- `ROADMAP.md` R5 blocks the next **stable** `studio-v2.*` tag until the GA gates (G1, G3, G4,
  G5) are met. 2.67.0 was tagged at the user's explicit request; a fork release moves no npm
  pointer, but the gate note stands for anyone reading the runbook.
- Five UI commits (spacing, ghost-row polish) ride the **next** tag; cutting 2.67.1 is a repeat
  of the runbook on torch.
- `checkedOn` in `internal/contract/pricing/catalog.go` is 2026-11-20; the 90-day gate goes red
  again around 2027-02.
- The 30-minute release-progress automation created for this handover was deleted on completion.
