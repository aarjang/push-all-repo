# Contract: External Commands (`git`, `gh`)

**Feature**: 001-bulk-repo-push

Every external call goes through one runner that sets a hardened environment and a context with timeout. Nothing else in the code base spawns processes.

## Runner environment (all `git` calls)

| Variable | Value | Purpose |
|----------|-------|---------|
| `GIT_TERMINAL_PROMPT` | `0` | Never block on username/password prompt |
| `GIT_ASKPASS` | path to a no-op helper (or `echo` on Unix, `cmd /c exit 1` on Windows) | Same, for helpers that bypass the terminal check |
| `GIT_SSH_COMMAND` | `ssh -o BatchMode=yes` (only if user has not set their own) | Fail fast when no key/agent |
| `LC_ALL` | `C` | Stable, parseable output |
| `GIT_OPTIONAL_LOCKS` | `0` | Avoid index lock contention with editors |

Working directory is always passed with `-C <path>`. Per-call timeout: 15 s for inspection commands, `--timeout` (default 120 s) for push and `gh repo create`.

## Inspection commands (read-only)

| Purpose | Command | Success interpretation | Failure interpretation |
|---------|---------|------------------------|------------------------|
| Git available & version | `git --version` | parse `git version X.Y` ; require ≥ 2.20 | exit 2 "git not found" |
| Is top-level repo | `git -C P rev-parse --show-toplevel` | output (path-cleaned) == P → `IsRepo`; ≠ P → `NestedRepo` | non-zero → `NotARepo` |
| Current branch | `git -C P symbolic-ref --quiet --short HEAD` | branch name | non-zero → detached |
| Upstream | `git -C P rev-parse --abbrev-ref --symbolic-full-name @{u}` | `remote/branch` | non-zero → no upstream |
| Remotes | `git -C P remote` | one name per line | empty → no remote |
| Ahead count | `git -C P rev-list --count @{u}..HEAD` | integer | only run when upstream exists |
| Dirty | `git -C P status --porcelain=v1 --untracked-files=normal` | lines → `DirtyFiles` | — |
| In-progress op | `git -C P rev-parse --git-path MERGE_HEAD` / `rebase-merge` / `rebase-apply` / `CHERRY_PICK_HEAD`, then `os.Stat` each | exists → `merge` / `rebase` / `rebase` / `cherry-pick` | — |
| Identity | `git -C P config user.name`, `git -C P config user.email` | both non-empty → `IdentityOK` | — |

## Mutating commands

| Purpose | Command | Notes |
|---------|---------|-------|
| Stage everything | `git -C P add -A` | Honours `.gitignore` |
| Commit | `git -C P commit -m MSG` | No `--no-verify`; hooks run |
| Push (has upstream) | `git -C P push` | Uses branch's configured upstream |
| Push (no upstream) | `git -C P push --set-upstream origin BRANCH` | Only when a remote named `origin` exists; otherwise the no-remote flow |

Forbidden anywhere in the code base: `push --force`, `push --force-with-lease`, `push --delete`, `reset`, `rebase`, `checkout`, `switch`, `clean`, `stash`.

## Push error classification (stderr, case-insensitive substring match, first hit wins)

| Pattern | Reason |
|---------|--------|
| `permission denied`, `authentication failed`, `could not read username`, `could not read password`, `publickey`, `403` | `AuthFailed` |
| `rejected`, `non-fast-forward`, `fetch first`, `failed to push some refs` | `Rejected` |
| `could not resolve host`, `connection timed out`, `connection refused`, `network is unreachable`, `unable to access`, `ssh: connect to host` | `NetworkError` |
| context deadline exceeded | `Timeout` |
| context cancelled | `Interrupted` |
| anything else | `GitError` with the last non-empty stderr line as `Detail` |

## GitHub CLI

| Purpose | Command | Interpretation |
|---------|---------|----------------|
| Installed | look up `gh` on `PATH` | missing → feature unavailable, warn once on stderr |
| Logged in | `gh auth status --hostname github.com` | exit 0 → logged in |
| Create + push | `gh -C P repo create NAME --private\|--public --source . --remote origin --push` | exit 0 → `Pushed` with `CreatedRemote` from `gh repo view --json url -q .url`; stderr containing `already exists` → re-prompt for name (FR-008b); other → `CreateRepoFailed` |

`gh` inherits the same runner environment, so it also never prompts.
