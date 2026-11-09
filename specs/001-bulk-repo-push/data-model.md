# Data Model: Bulk Repo Push

**Feature**: 001-bulk-repo-push | **Date**: 2026-11-09

All data lives in memory for the duration of one run. The only persisted artifact is the optional Markdown report (see [contracts/report-format.md](contracts/report-format.md)). Types below map to Go structs in `internal/model`.

## Entities

### Run

One execution of the tool.

| Field | Type | Notes |
|-------|------|-------|
| Mode | enum `Mode` | `FolderOfProjects` or `SingleProject` |
| RootPath | string | Absolute, cleaned path the user provided |
| StartedAt | time | Local time |
| FinishedAt | time | Set when the loop ends or is interrupted |
| Interrupted | bool | True when Ctrl+C ended the run early |
| Options | `Options` | Resolved flags (timeout, strict, no-color, verbose, default answers) |
| Results | []`Result` | One per Candidate, in processing order |
| ToolVersion | string | For the report header |

**Validation**: RootPath must exist, be a directory, and be readable (FR-003). In `SingleProject` mode RootPath must itself be a Git repository top-level.

### Candidate

A folder examined during discovery.

| Field | Type | Notes |
|-------|------|-------|
| Path | string | Absolute |
| Name | string | Folder base name; display name |
| IsRepo | bool | `git rev-parse --show-toplevel` succeeded and equals Path |
| Branch | string | Empty when detached |
| Upstream | string | e.g. `origin/main`; empty when none |
| Remotes | []string | From `git remote` |
| AheadCount | int | Commits on Branch not on Upstream; `-1` when no upstream |
| Dirty | bool | `git status --porcelain` non-empty |
| DirtyFiles | []string | Porcelain lines, shown before the FR-007 prompt |
| InProgressOp | string | `merge`, `rebase`, `cherry-pick`, or empty |
| IdentityOK | bool | `user.name` and `user.email` both set (needed only if committing) |
| Excluded | bool | Set by the user in the review step (FR-017) |

**Derived**: `NeedsPush = IsRepo && Branch != "" && (Upstream == "" || AheadCount > 0)`.

**Ordering rule**: root folder first if it is a repo, then children sorted case-insensitively by Name (FR-005).

### Decision

An answer the user gave at a prompt during the run.

| Field | Type | Notes |
|-------|------|-------|
| Kind | enum | `DirtyChoice`, `CreateRepo`, `SaveReport` |
| DirtyChoice | enum | `CommitAndPush`, `PushOnly`, `Skip` |
| CommitMessage | string | Only for `CommitAndPush` |
| CreateRepo | bool | Accept or decline the GitHub offer |
| RepoName | string | Defaults to Candidate.Name |
| Visibility | enum | `Private` (default) or `Public` |
| ApplyToAll | bool | Remember for remaining candidates (FR-007a); applies to `DirtyChoice` and to `CreateRepo` accept/decline + visibility, never to RepoName |

### Result

The outcome for one Candidate.

| Field | Type | Notes |
|-------|------|-------|
| Candidate | `Candidate` | |
| Status | enum `Status` | See states below |
| Reason | enum `Reason` | Required when Status is `NotPushed` or `NotProcessed`; empty otherwise |
| Detail | string | Last meaningful stderr line or extra context, trimmed to 200 chars |
| Committed | bool | Tool created a commit |
| CommitMessage | string | When Committed |
| CreatedRemote | string | URL of the GitHub repo created, when any |
| Duration | duration | Wall time for this candidate |

### Report

| Field | Type | Notes |
|-------|------|-------|
| Run | `Run` | |
| Groups | map Status → []Result | Stable order: Pushed, UpToDate, NotPushed, SkippedByUser, NotProcessed |
| Counts | map Status → int | FR-012 |
| ExitCode | int | Derived; see rules below |

## Enumerations

### Status

| Value | Meaning |
|-------|---------|
| `Pushed` | Push succeeded (includes "new GitHub repository created", flagged by `CreatedRemote`) |
| `UpToDate` | Repo had nothing to push (AheadCount == 0, not dirty or user chose PushOnly) |
| `NotPushed` | Attempted or evaluated and could not push; `Reason` set |
| `SkippedByUser` | Excluded in review, or answered `Skip` at the dirty prompt |
| `NotProcessed` | Run was interrupted before reaching it |

### Reason

| Value | Human text in report |
|-------|---------------------|
| `NotARepo` | not a Git repository |
| `NestedRepo` | inside another repository (not a top-level project) |
| `NoRemote` | no remote configured |
| `NoRemoteLoginRequired` | no remote (GitHub login required to create one) |
| `NoRemoteDeclined` | no remote (user declined to create one) |
| `CreateRepoFailed` | could not create GitHub repository: <detail> |
| `DetachedHead` | no current branch (detached HEAD) |
| `OperationInProgress` | merge/rebase/cherry-pick in progress |
| `IdentityMissing` | cannot commit: Git identity not configured |
| `CommitFailed` | commit failed: <detail> |
| `AuthFailed` | authentication failed |
| `Rejected` | push rejected by remote (fetch first / non-fast-forward) |
| `NetworkError` | could not reach remote |
| `Timeout` | push exceeded <N> s |
| `Interrupted` | interrupted by user |
| `GitError` | git error: <detail> |
| `AnswerRequired` | answer required (no terminal): pass <flag> |

## State transitions (per Candidate)

```text
Discovered
  ├─ !IsRepo ──────────────────────────────▶ NotPushed(NotARepo)
  ├─ nested ───────────────────────────────▶ NotPushed(NestedRepo)
  ├─ Excluded (review step) ───────────────▶ SkippedByUser
  ├─ Branch == "" ─────────────────────────▶ NotPushed(DetachedHead)
  ├─ InProgressOp != "" ───────────────────▶ NotPushed(OperationInProgress)
  ├─ Dirty ─▶ prompt FR-007
  │     ├─ Skip ───────────────────────────▶ SkippedByUser
  │     ├─ CommitAndPush
  │     │     ├─ !IdentityOK ─────────────▶ NotPushed(IdentityMissing)
  │     │     ├─ commit fails ────────────▶ NotPushed(CommitFailed)
  │     │     └─ commit ok ─▶ (continue as clean, Committed=true)
  │     └─ PushOnly ─▶ (continue as clean)
  ├─ len(Remotes)==0 ─▶ offer FR-008
  │     ├─ gh unavailable/not logged in ──▶ NotPushed(NoRemoteLoginRequired)
  │     ├─ declined ──────────────────────▶ NotPushed(NoRemoteDeclined)
  │     ├─ create fails (incl. name exists after re-prompt declined) ▶ NotPushed(CreateRepoFailed)
  │     └─ created ─▶ push via gh --push ─▶ Pushed(CreatedRemote=url) | NotPushed(<push reason>)
  ├─ Upstream != "" && AheadCount == 0 ────▶ UpToDate
  └─ push (set-upstream if none)
        ├─ ok ──────────────────────────────▶ Pushed
        ├─ auth / rejected / network / timeout / interrupted ▶ NotPushed(reason)
        └─ other ───────────────────────────▶ NotPushed(GitError)
Any candidate not yet reached when interrupted ─▶ NotProcessed(Interrupted)
```

## Exit code rules

| Condition | Code |
|-----------|------|
| Every result is `Pushed` or `UpToDate` (SkippedByUser allowed unless `--strict`) | 0 |
| Any `NotPushed`, or any `SkippedByUser` with `--strict` | 1 |
| Invalid flags, path missing/unreadable, single-project path is not a repo, `git` not found | 2 |
| Interrupted by user | 130 |
