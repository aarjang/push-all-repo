# Quickstart: Validating Bulk Repo Push

**Feature**: 001-bulk-repo-push

This guide proves the feature end-to-end without touching any real remote until the last, optional scenario. Contracts: [cli.md](contracts/cli.md), [report-format.md](contracts/report-format.md), [git-commands.md](contracts/git-commands.md). Entities and statuses: [data-model.md](data-model.md).

## Prerequisites

- Go 1.24+ (`go version`)
- Git ≥ 2.20 (`git --version`) with `user.name` and `user.email` configured
- Optional, for scenario 6 only: GitHub CLI logged in (`gh auth status`)

## Build and run tests

```bash
go build ./cmd/pushall
```

```bash
go test ./...
```

Expected: build produces `./pushall` (or `pushall.exe`); all unit and integration tests pass on the current OS.

## Fixture: a folder of projects with local remotes

The integration tests build this automatically; for manual runs, create it once (bash on Linux/macOS/Git Bash; PowerShell equivalents in the README):

```bash
mkdir -p /tmp/pa/remotes /tmp/pa/projects && cd /tmp/pa/projects && \
for r in alpha beta gamma; do git init -q --bare ../remotes/$r.git; git init -q $r && (cd $r && git commit -q --allow-empty -m init && git remote add origin ../../remotes/$r.git && git push -q -u origin HEAD); done && \
(cd alpha && git commit -q --allow-empty -m "unpushed") && \
(cd beta && echo x > dirty.txt) && \
git init -q delta && (cd delta && git commit -q --allow-empty -m init) && \
mkdir notes && echo hi > notes/readme.txt
```

Resulting folders: `alpha` (1 unpushed commit), `beta` (up to date, dirty), `gamma` (up to date), `delta` (repo, no remote), `notes` (not a repo).

## Scenario 1 — Folder mode, happy path (Story 1)

```bash
./pushall --mode folder /tmp/pa/projects --dirty push-only --create-repo no --no-review --report -
```

Expected:
- Progress shows 5 entries processed in order alpha, beta, delta, gamma, notes.
- Report: Pushed (1) alpha · Already up to date (2) beta "(uncommitted changes left untouched)", gamma · Not pushed (1) delta "no remote (user declined to create one)" · Not a Git repository: notes.
- `echo $?` → `1` (delta not pushed).
- `git -C /tmp/pa/remotes/alpha.git log --oneline | head -1` shows "unpushed".

## Scenario 2 — Single project mode (Story 2)

```bash
(cd /tmp/pa/projects/gamma && git commit -q --allow-empty -m "second") && ./pushall --mode single /tmp/pa/projects/gamma --report -
```

Expected: one-entry report, gamma pushed, exit 0. Then:

```bash
./pushall --mode single /tmp/pa/projects/notes --report -
```

Expected: error "not a Git repository", exit 2, no report.

## Scenario 3 — Interactive dirty prompt (FR-007)

```bash
./pushall --mode folder /tmp/pa/projects --create-repo no --no-review --report -
```

At `beta`, the tool lists `?? dirty.txt` and asks Commit / Push only / Skip. Choose **Commit all changes and push**, accept the default message, answer **No** to "apply to all". Expected: beta appears under Pushed with `committed by tool: "chore: bulk push on …"`; `git -C /tmp/pa/remotes/beta.git log --oneline | head -1` shows that message.

Re-run and choose **Skip**: beta appears under Skipped by user; exit 0 without `--strict`, exit 1 with `--strict`.

## Scenario 4 — Review and exclude (Story 3)

```bash
./pushall --mode folder /tmp/pa/projects --dirty push-only --create-repo no --report -
```

In the review list uncheck `alpha`, confirm. Expected: alpha under Skipped by user with "excluded in review"; nothing pushed for alpha.

## Scenario 5 — Failure isolation and classification

Break one remote and rerun:

```bash
(cd /tmp/pa/projects/alpha && git commit -q --allow-empty -m "again" && git remote set-url origin /tmp/pa/does-not-exist.git)
```

```bash
./pushall --mode folder /tmp/pa/projects --dirty push-only --create-repo no --no-review --report ./report.md
```

Expected: alpha under Not pushed with a `git error` / `could not read` reason, every other repo still processed, `report.md` written with identical entries to the screen, exit 1.

Detached HEAD and merge-in-progress:

```bash
(cd /tmp/pa/projects/gamma && git checkout -q --detach) && ./pushall --mode folder /tmp/pa/projects --dirty push-only --create-repo no --no-review --report - ; (cd /tmp/pa/projects/gamma && git checkout -q -)
```

Expected: gamma under Not pushed "no current branch (detached HEAD)".

## Scenario 6 — Create GitHub repository (FR-008, real `gh`, optional)

Requires `gh auth status` to succeed. This creates a real private repository under your account; delete it afterwards.

```bash
./pushall --mode single /tmp/pa/projects/delta --report -
```

Expected: offer "Create GitHub repository `delta` (private) and push?". Accept. Report: delta Pushed "pushed to new GitHub repository https://github.com/<you>/delta (private)". Clean up with `gh repo delete <you>/delta --yes`.

With `gh` logged out (`gh auth logout`), the same command reports "no remote (GitHub login required to create one)" and exit 1.

## Scenario 7 — Interrupt (FR-018)

Run scenario 1 and press Ctrl+C while a push is in flight (make one remote slow or large). Expected: current repo reported "interrupted by user", remaining repos under Not processed, report still printed, exit 130.

## Scenario 8 — Cross-platform parity (SC-005)

CI runs `go test ./...` on ubuntu, macos and windows. The integration test `TestReportParity` renders the Markdown for a fixed fixture and compares it byte-for-byte against a golden file; the golden must pass on all three runners.

## Cleanup

```bash
rm -rf /tmp/pa
```

## Validation log

**2026-11-09, macOS (darwin/arm64), Go 1.24.7, git 2.54.0, gh 2.100.0**

| Scenario | Result | How verified |
|----------|--------|--------------|
| Build and tests | PASS | `go vet ./...`, `go test -race ./...` all green (unit + 30 real-git integration tests) |
| 1 Folder mode happy path | PASS | Manual run of the built binary on the fixture: gamma pushed, beta "up to date (uncommitted changes left untouched)", delta "no remote (user declined to create one)", notes under "Not a Git repository", exit 1. Also `TestFolderModeHappyPath` |
| 2 Single project | PASS | Manual: gamma pushed/up to date with one-entry report, exit 0; `notes` → "not a Git repository", exit 2, no report. Also `TestSingleMode*` |
| 3 Dirty prompt | PASS (scripted) | `TestDirtyCommitAndPush`, `TestDirtyPushOnly`, `TestDirtySkip` (+ `--strict`), `TestDirtyApplyToAll`, `TestDirtyCommitWithoutIdentity`. The huh form itself was not driven by hand in this session |
| 4 Review and exclude | PASS (scripted) | `TestReviewExcludes`, `TestNoReviewFlagSkipsPrompt`, `TestDeclineConfirmStart` |
| 5 Failure isolation and report file | PASS | Manual: alpha with broken remote → "could not reach remote", every other repo processed, `report.md` written with identical entries, exit 1; detached gamma → "no current branch (detached HEAD)". Also `TestFailureIsolation`, `TestDetachedHead`, `TestMergeInProgress`, `TestRejectedPush`, `TestReport*` |
| 6 Create GitHub repository (real gh) | NOT RUN | Creates a real repository on the user's account; left for the user. Flow covered by `TestNoRemote*` with a fake client (accept, decline, name taken, retry, not logged in, gh missing) |
| 7 Interrupt | PASS (test) | `TestInterruptMidRun`: Ctrl+C mid-push → current repo "interrupted by user", rest "Not processed", report printed, exit 130. `TestPushTimeout` covers "push exceeded 1 s" |
| 8 Cross-platform parity | PARTIAL | Golden text and Markdown tests pass on macOS; Linux and Windows runs happen in the CI matrix once the branch is pushed |

Extra findings fixed during validation: flags may now appear before or after PATH; a prompt that cannot be shown without a terminal is reported as "answer required (no terminal): pass --dirty …" instead of a git error.
