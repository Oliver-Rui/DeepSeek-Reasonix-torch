---
owner: @SivanCola
backup: @esengine
status: active
reviewed: 2026-11-19
---

# Branches

- `torch` is the default branch: the complete application, the release line, and the one branch that must always build.
- `studio` is a read-only snapshot of upstream 2.61.0.
- Each `area/*` branch is a module workstream that starts at the `torch` tip, so pulling any of them gives a complete application rather than a partial tree.

```
studio   upstream 2.61.0 snapshot (read-only)
torch    integration and releases (default)
area/core         area/kernel       area/collab
area/settings     area/workbench    area/sidebar
area/conversation area/chrome       area/docs
```

## Which branch does my change belong to?

The path table lives in `tools/branchmap/areas.go`. Ask it instead of guessing:

```bash
make branch-for                 # the working tree's changed and untracked files
go run ./tools/branchmap <path> # one or more explicit paths
go run ./tools/branchmap --list # the full table
```

Unmatched paths are reported and assigned to `area/core`; add a rule rather than
letting a feature test fall into the shared commit.

The areas follow the component ownership that exists today:

| Branch | Owns |
|---|---|
| `area/core` | `src/port`, `src/i18n`, `src/state`, `src/styles`, `src/design`, `src/boot`, shared hooks and primitives |
| `area/kernel` | `internal/`, `cmd/`, Go files under `desktop/`, module files |
| `area/collab` | feedback, peers, dispatch board, remote, memory |
| `area/settings` | settings pages, providers, models, extensions, account, market |
| `area/workbench` | workbench, files, browser, terminal, document and chart preview |
| `area/sidebar` | sidebar, projects, session rows, palette and navigation |
| `area/conversation` | pane, transcript, composer, plans, cards, panels |
| `area/chrome` | app shell, tabs, rails, toasts, updates, welcome |
| `area/docs` | docs, release notes, developer tools, CI, README |

## Workflow

### One module

```bash
make branch-for
# area/sidebar  desktop/frontend-next/src/ui/Sidebar.tsx
git switch area/sidebar
git pull --ff-only
# edit, test
git commit -m "sidebar: ..."
git push
git switch torch
git merge --ff-only area/sidebar
```

### Shared code

`src/port`, `src/i18n`, `src/state`, `src/styles`, and shared primitives go to
`area/core`. Because every module branch starts at the `torch` tip, a shared
change merged into `torch` can be fast-forwarded into the other area branches
when they have no unique commits.

### More than one module

If `make branch-check` reports more than one area, do not force the change into
one of them. Open `feat/<topic>` from `torch`, split it into a `core` commit
plus one commit per module, or make one complete cross-area commit on `torch`.
After it lands, rebase or fast-forward the affected `area/*` branches.

### Kernel and docs

`area/kernel` for the Go kernel and command entrypoints. `area/docs` for
`docs/`, `release-notes/`, `tools/`, CI, README and CONTRIBUTING.

## Commit rule

One commit expresses one complete final behavior. Squash incremental fixes
before pushing; never leave a `fix the previous commit` chain on a shared
branch. A layout change must arrive with its styles, tests and translations in
the same commit, so a person who pulls that commit does not get half a change.

Cross-area commits are the exception, not the default: when a change genuinely
needs the kernel and two UI modules, it belongs on `torch` as one reviewed
commit, and the area branches are fast-forwarded to it.

## Pulling a module

- `area/<module>` is a full application. It includes every other module at the time it was branched, plus the module-scoped commits made on it.
- A module branch that omits shared code or a sibling module's API cannot build, so the branch is deliberately cumulative.
- The consolidated history on `torch` has one final-state commit per module.
- Those module commits are review snapshots, not standalone builds: the shared `port` contract changed, so a final core tree beside a 2.61 UI cannot type-check. Check out `torch` or an `area/*` branch to build.
- Use `git log --oneline torch` and the `core:`, `kernel:`, `sidebar:` prefixes to review a module's complete 2.61 → current change.
- The old incremental history is not part of the public line; a local bundle/mirror holds it for rollback, and the upstream repository remains the 2.61.0 reference.
