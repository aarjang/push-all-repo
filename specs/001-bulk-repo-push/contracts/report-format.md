# Contract: Report Format

**Feature**: 001-bulk-repo-push

The same data renders in two forms. Content (entries, statuses, reasons, counts) MUST be identical on all operating systems for the same input (SC-005); only colouring differs.

## On-screen (text)

```text
Push report — 2026-11-09 14:03:17 — /Users/me/projects — folder mode

Pushed (3)
  ✔ api-server          main      2 commits pushed
  ✔ website             develop   1 commit pushed (committed: "chore: bulk push on 2026-11-09 14:03")
  ✔ new-tool            main      pushed to new GitHub repository github.com/me/new-tool (private)

Already up to date (2)
  = dotfiles            main
  = notes               main      (uncommitted changes left untouched)

Not pushed (2)
  ✘ legacy-app          main      authentication failed
  ✘ scratch             —         no current branch (detached HEAD)

Skipped by user (1)
  – experiments         main      excluded in review

Not a Git repository: Downloads, screenshots

Summary: 3 pushed, 2 up to date, 2 not pushed, 1 skipped by user · 8 folders · 41 s
```

Rules:
- Groups appear in fixed order and are omitted when empty, except that `Not pushed` always appears when count > 0.
- Folders that are not repositories are listed on one line under `Not a Git repository:` and counted in "folders" but not in "not pushed". (They still produce `NotPushed(NotARepo)` results internally so the Markdown file lists them; the text view compacts them.)
- `Not processed (N)` group appears only after an interruption.
- Symbols: `✔ = – ✘ ○`; in `--no-color`/non-UTF-8 consoles they become `+ = - x o`.
- Column widths adapt to the longest name, capped at 40 characters (longer names are truncated with `…`).

## Markdown file

Default path: `./push-report-YYYYMMDD-HHMMSS.md` (local time). Structure:

```markdown
# Push report

- **Date**: 2026-11-09 14:03:17
- **Root**: /Users/me/projects
- **Mode**: folder of projects
- **Tool**: pushall v0.1.0
- **Duration**: 41 s
- **Summary**: 3 pushed · 2 up to date · 2 not pushed · 1 skipped by user · 8 folders

## Pushed (3)

| Repository | Branch | Details |
|------------|--------|---------|
| api-server | main | 2 commits pushed |
| website | develop | 1 commit pushed; committed by tool: "chore: bulk push on 2026-11-09 14:03" |
| new-tool | main | pushed to new GitHub repository https://github.com/me/new-tool (private) |

## Already up to date (2)
...

## Not pushed (2)

| Repository | Branch | Reason |
|------------|--------|--------|
| legacy-app | main | authentication failed |
| scratch | — | no current branch (detached HEAD) |

## Skipped by user (1)
...

## Not a Git repository (2)

Downloads, screenshots
```

Rules:
- Every repository appears exactly once.
- `Reason` text is the human string from the `Reason` enumeration in [data-model.md](../data-model.md); `Detail` is appended after a colon when present.
- Timestamps are local time in `YYYY-MM-DD HH:MM:SS`.
- The file is written with UTF-8 and `\n` line endings on all OSes.
