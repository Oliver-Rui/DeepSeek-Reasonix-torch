---
owner: @SivanCola
backup: @esengine
status: active
reviewed: 2026-11-19
---

# Branches

There is one branch: `torch`. It holds the complete application, the release
line, and every change. Nothing is split by module any more — a module branch
that carries only its own files cannot build, and a reader who pulls it gets a
partial tree.

```
torch    the complete application, releases, and the default branch
```

## Workflow

```bash
git switch torch
git pull --ff-only
# edit, test
git commit -m "<area>: ..."
git push
```

Work directly on `torch`. When a change is too large to review as one commit,
use a short-lived topic branch off `torch` and merge it back; do not leave the
topic branch on the remote once it has landed.

## Commit messages

Start the subject with the area the change is mostly about, so `git log` reads
as a map of the tree:

| Prefix | Area |
|---|---|
| `core:` | `src/port`, `src/i18n`, `src/state`, `src/styles`, `src/design`, shared hooks |
| `kernel:` | `internal/`, `cmd/`, Go files under `desktop/`, module files |
| `settings:` | settings pages, providers, models, extensions, account, market |
| `workbench:` | workbench, files, browser, terminal, document and chart preview |
| `sidebar:` | sidebar, projects, session rows, palette and navigation |
| `conversation:` | pane, transcript, composer, plans, cards, panels |
| `chrome:` | app shell, tabs, rails, toasts, updates, welcome |
| `docs:` | `docs/`, release notes, developer tools, CI, README |
| `perf:` | a change whose point is load or latency |

## Commit rule

One commit expresses one complete final behavior. Squash incremental fixes
before pushing; never leave a `fix the previous commit` chain on `torch`. A
layout change must arrive with its styles, tests and translations in the same
commit, so a person who pulls that commit does not get half a change.

## The path map

`tools/branchmap` still classifies a path by area, but it now answers "which
prefix does this commit want", not "which branch":

```bash
make branch-for                 # the working tree's changed and untracked files
go run ./tools/branchmap <path> # one or more explicit paths
go run ./tools/branchmap --list # the full table
```

A change that spans areas takes the prefix of the area it is mostly about and
says the rest in the body.

## History

`torch` is a rebuilt line: a 2.61.0 snapshot commit, then one final-state commit
per module through 2.66.1, then the work since. Those module commits are review
snapshots, not standalone builds — the shared `port` contract changed, so a
final core tree beside a 2.61 UI cannot type-check. Check out `torch` to build.

The pre-rebuild incremental history is not part of the public line; a local
bundle and mirror hold it for rollback, and the upstream repository remains the
2.61.0 reference.
