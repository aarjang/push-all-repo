# Research: Bulk Repo Push

**Feature**: 001-bulk-repo-push | **Date**: 2026-11-09

All unknowns from the Technical Context are resolved below. Each entry records the decision, why, and what else was considered.

## R1. Implementation language and runtime

**Decision**: Go (1.24+), compiled to a single static binary per OS/arch.

**Rationale**:
- The spec requires identical behaviour on Linux, macOS and Windows (FR-001) and publication on GitHub (FR-019). Go cross-compiles from one machine to all three targets with no runtime dependency, so users download one file and run it.
- Spawning and reading `git` processes, walking directories, handling signals (Ctrl+C, FR-018) and path quirks on Windows are all first-class in the standard library.
- Go 1.24.7 is already installed on the development machine.

**Alternatives considered**:
- *Python*: fastest to write, but users need a matching interpreter; Windows distribution (PyInstaller) is heavy and fragile; pipx is an extra step for a "simple" tool.
- *Node.js / TypeScript*: same runtime-dependency problem; single-file bundling (pkg, SEA) is still immature on Windows.
- *Rust*: equally good binaries, but slower iteration for a tool of this size and not installed here.

## R2. How to talk to Git

**Decision**: Shell out to the user's installed `git` executable (via `os/exec`) with plumbing commands and machine-readable flags. Do not embed a Git implementation.

**Rationale**:
- FR-014 requires reusing the user's existing credentials: SSH agent, `~/.ssh/config`, credential helpers (osxkeychain, wincred, libsecret), `gh auth` git-credential integration. The system `git` honours all of these for free; a library would have to re-implement each one.
- Hooks, `includeIf` configs, signing, and proxy settings keep working.
- Every operation needed is a stable plumbing command (see `contracts/git-commands.md`).

**Alternatives considered**:
- *go-git (pure Go)*: push authentication must be wired by hand per transport; credential helpers are unsupported; SSH needs explicit key loading. Rejected.
- *libgit2 bindings (git2go)*: CGO, platform build pain, same credential problem. Rejected.

**Non-interactive safety**: every `git` invocation runs with `GIT_TERMINAL_PROMPT=0`, `GIT_ASKPASS` pointed to a no-op, and `GIT_SSH_COMMAND="ssh -o BatchMode=yes"` so a missing credential fails fast with "authentication failed" instead of hanging (edge case in spec). Each push also has a configurable timeout (default 120 s).

## R3. Creating a GitHub repository (FR-008)

**Decision**: Shell out to the GitHub CLI: `gh auth status` to detect a login, `gh repo create <name> --private|--public --source <path> --remote origin --push`. If `gh` is missing or not logged in, skip creation and report "GitHub login required".

**Rationale**:
- FR-014 forbids asking the user for tokens. `gh` already holds the user's login on all three OSes and is the official GitHub tool.
- `gh repo create --source --remote origin` creates the repo, adds the remote, and pushes in one well-tested command; FR-008b (name already exists) surfaces as a clear `gh` error we pass through.
- `gh` 2.100 is installed here; absence is handled gracefully.

**Alternatives considered**:
- *GitHub REST API with token from `gh auth token`*: works but duplicates what `gh repo create` does and adds HTTP, JSON and error-mapping code. Rejected for v1.
- *Ask user for a token*: violates FR-014. Rejected.

## R4. Interactive prompts and progress output

**Decision**: `github.com/charmbracelet/huh` for prompts (select, confirm, multi-select, text input) and `github.com/charmbracelet/lipgloss` for light styling of the progress log and report. Fall back to plain text when stdout is not a TTY or `--no-color` is set.

**Rationale**:
- Spec asks for "simple" selection (FR-002, FR-003, FR-017, FR-007). `huh` gives arrow-key menus, checkboxes for the review step, and validated text input out of the box, and renders correctly on Windows Terminal, cmd.exe (Windows 10+), macOS Terminal and Linux terminals.
- Pure Go, no CGO, widely used, MIT licensed.

**Alternatives considered**:
- *survey (AlecAivazis)*: in maintenance mode. Rejected.
- *bubbletea full TUI*: more power than needed; a sequential prompt flow is clearer for "process repos one at a time". Rejected.
- *Plain `fmt.Scanln` prompts*: zero dependencies but poor UX for multi-select exclusion (FR-017). Rejected.

## R5. CLI argument handling

**Decision**: Standard library `flag` package. Flags are optional shortcuts; with no flags the tool runs fully interactive.

**Rationale**: Only a handful of flags are needed (see `contracts/cli.md`); cobra/urfave add weight for no benefit. Flags let scripts skip prompts (spec assumption: "nice-to-have"), and `--yes`-style flags map to the "apply to all" answers in FR-007a.

## R6. Repository state detection

**Decision**: Use these plumbing commands per candidate (full list in `contracts/git-commands.md`):

| Question | Command | Interpretation |
|----------|---------|----------------|
| Is this a Git repo (top-level)? | `git -C <p> rev-parse --show-toplevel` | Success and output equals `<p>` → yes; different path → nested inside another repo → not a candidate |
| Current branch | `git symbolic-ref --quiet --short HEAD` | Non-zero exit → detached HEAD → skip |
| Upstream | `git rev-parse --abbrev-ref --symbolic-full-name @{u}` | Non-zero exit → no upstream |
| Remotes | `git remote` | Empty → no remote |
| Unpushed commits | `git rev-list --count @{u}..HEAD` | >0 → needs push; no upstream → needs push |
| Uncommitted changes | `git status --porcelain=v1 --untracked-files=normal` | Any line → dirty |
| Operation in progress | `git rev-parse --git-path MERGE_HEAD`, `rebase-merge`, `rebase-apply`, `CHERRY_PICK_HEAD` exist | Any exists → skip with reason |

**Rationale**: All are stable, scriptable, locale-independent (`--porcelain`), and avoid parsing human-facing output.

## R7. Discovery scope and ordering

**Decision**: Read the direct children of the root folder with `os.ReadDir`, keep directories only (follow symlinked directories, skip hidden dot-folders), sort case-insensitively by name, and prepend the root itself if it is a repository. No recursion.

**Rationale**: Matches FR-004/FR-005 and the spec assumptions. Skipping dot-folders avoids `.git`, `.idea`, `.vscode`. Case-insensitive sort gives the same order on macOS (case-insensitive FS) and Linux.

## R8. Committing on the user's behalf (FR-007)

**Decision**: On "commit and push": `git add -A` (respects `.gitignore`) then `git commit -m <msg>`. Default message: `chore: bulk push on 2026-11-09 14:03` (local time). Before committing, check `git config user.name` and `user.email`; if either is empty, report "cannot commit: Git identity not configured" and offer "push only" or "skip" instead. Show the `git status --porcelain` file list before asking.

**Rationale**: `add -A` is the only staging mode that matches the spec wording ("all tracked changes and non-ignored new files"). Never use `--no-verify`; hooks are the user's policy.

## R9. Timeouts, cancellation and partial reports (FR-018)

**Decision**: A `context.Context` cancelled on SIGINT/SIGTERM (`signal.NotifyContext`) is passed to every `exec.CommandContext`. On cancellation the in-flight git process is killed, its repository is recorded as "not pushed: interrupted by user", remaining candidates are recorded as "not processed", and the report is still printed. A second Ctrl+C exits immediately.

**Rationale**: Standard Go idiom; works identically on Windows (console Ctrl+C event) and Unix.

## R10. Report format and exit codes

**Decision**:
- On-screen report: grouped sections (Pushed / Already up to date / Not pushed / Skipped by user / Not processed) with a summary line. Markdown file on request, default name `push-report-YYYYMMDD-HHMMSS.md` in the current working directory.
- Exit codes: `0` everything pushed or up to date; `1` at least one repository not pushed (including user-skipped if `--strict`); `2` invalid arguments or unreadable path; `130` interrupted.

**Rationale**: FR-011, FR-012, FR-013, FR-016. Markdown renders on GitHub and in any editor. Codes follow Unix conventions.

## R11. Testing strategy

**Decision**: `go test` with the standard `testing` package plus `github.com/stretchr/testify` for assertions.
- *Unit*: discovery ordering, status parsing, report rendering, exit-code mapping, flag parsing.
- *Integration*: tests create temporary repositories and **bare repositories on the local filesystem** used as `file://` remotes, so push/no-upstream/rejected-push/dirty/detached/merge-in-progress cases run offline and on every OS in CI.
- *GitHub creation*: the `gh` wrapper is behind an interface; tests use a fake. A manual quickstart scenario covers the real `gh`.
- *CI*: GitHub Actions matrix `ubuntu-latest`, `macos-latest`, `windows-latest` running `go vet`, `go test ./...`, and a cross-compile check.

**Rationale**: Real `git` binaries exercised on all three OSes catch path and line-ending bugs that mocks hide; no network is needed.

## R12. Distribution

**Decision**: GoReleaser via GitHub Actions on tag push → GitHub Releases with archives for linux/darwin/windows × amd64/arm64, plus `go install github.com/aarjang/push-all-repo/cmd/pushall@latest`. README documents both.

**Rationale**: FR-019; zero-install download for non-Go users, one command for Go users. Homebrew tap / Scoop / winget are deferred.
