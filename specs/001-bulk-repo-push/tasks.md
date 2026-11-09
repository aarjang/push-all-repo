---

description: "Task list for Bulk Repo Push"
---

# Tasks: Bulk Repo Push

**Input**: Design documents from `/specs/001-bulk-repo-push/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, contracts/report-format.md, contracts/git-commands.md, quickstart.md

**Tests**: Included. plan.md (research R11) defines the test strategy as part of the design: unit tests with `testing` + testify, and offline integration tests using temporary repositories with bare `file://` remotes. Test tasks are written before the implementation they cover and must fail first.

**Organization**: Tasks are grouped by user story so each story is an independently testable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on unfinished tasks)
- **[Story]**: User story label (US1–US4) for story-phase tasks only
- Every task names the exact file(s) it touches

## Path Conventions

Single Go module at the repository root (module path `github.com/aarjang/push-all-repo`). Binary in `cmd/pushall/`, logic in `internal/<pkg>/`, real-git integration tests in `tests/integration/`. See plan.md "Source Code" tree.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Go module, directory skeleton, dependencies, licence, CI.

- [X] T001 Initialise the Go module: run `go mod init github.com/aarjang/push-all-repo` and set `go 1.24` in `go.mod`
- [X] T002 Create the directory skeleton from plan.md with a `doc.go` (package comment) in each: `cmd/pushall/`, `internal/discover/`, `internal/gitx/`, `internal/github/`, `internal/model/`, `internal/ui/`, `internal/report/`, `internal/run/`, `tests/integration/`
- [X] T003 Add dependencies with `go get`: `github.com/charmbracelet/huh`, `github.com/charmbracelet/lipgloss`, `github.com/stretchr/testify`; commit the resulting `go.mod` and `go.sum`
- [X] T004 [P] Write `LICENSE` (MIT, copyright 2026 the repository owner) and extend `.gitignore` with Go build outputs (`/pushall`, `/pushall.exe`, `/dist/`, `*.test`, `coverage.out`)
- [X] T005 [P] Write `.github/workflows/ci.yml`: on push and pull_request, matrix `ubuntu-latest`/`macos-latest`/`windows-latest`, steps: checkout, setup-go 1.24, `git config --global user.name/user.email` (tests commit), `go vet ./...`, `go test ./...`, and a cross-compile step `GOOS=linux GOARCH=amd64 go build ./cmd/pushall` plus `GOOS=windows` and `GOOS=darwin`
- [X] T006 [P] Write `Makefile` with targets `build` (`go build -o pushall ./cmd/pushall`), `test` (`go test ./...`), `vet`, `lint` (`gofmt -l . && go vet ./...`), `release-dry` (`goreleaser release --snapshot --clean`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared types, the hardened `git` runner, repository inspection, error classification, UI primitives, report core, and the integration-test harness. Every user story depends on these.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Shared model

- [X] T007 Define types in `internal/model/model.go` exactly as data-model.md: `Mode` (`FolderOfProjects`, `SingleProject`); `Run` (Mode, RootPath, StartedAt, FinishedAt, Interrupted, Options, Results, ToolVersion); `Candidate` (Path, Name, IsRepo, Branch, Upstream, Remotes, AheadCount with `-1` meaning no upstream, Dirty, DirtyFiles, InProgressOp one of `merge`/`rebase`/`cherry-pick`/empty, IdentityOK, Excluded) with method `NeedsPush() bool` = `IsRepo && Branch != "" && (Upstream == "" || AheadCount > 0)`; `Decision` (Kind, DirtyChoice `CommitAndPush`/`PushOnly`/`Skip`, CommitMessage, CreateRepo, RepoName, Visibility `Private` default/`Public`, ApplyToAll); `Result` (Candidate, Status, Reason, Detail trimmed to 200 chars, Committed, CommitMessage, CreatedRemote, Duration); `Options` (Mode, Path, Dirty, Message, CreateRepo, Visibility, NoReview, ReportPath, Timeout default 120s, Strict, NoColor, Verbose)
- [X] T008 Define enums in `internal/model/status.go`: `Status` (`Pushed`, `UpToDate`, `NotPushed`, `SkippedByUser`, `NotProcessed`) with fixed display order, and `Reason` with every value from data-model.md (`NotARepo`, `NestedRepo`, `NoRemote`, `NoRemoteLoginRequired`, `NoRemoteDeclined`, `CreateRepoFailed`, `DetachedHead`, `OperationInProgress`, `IdentityMissing`, `CommitFailed`, `AuthFailed`, `Rejected`, `NetworkError`, `Timeout`, `Interrupted`, `GitError`) and a `Text() string` method returning the exact human strings in the data-model.md "Reason" table (e.g. `Timeout` → `push exceeded <N> s` using a format argument)
- [X] T009 [P] Unit-test `internal/model/model_test.go`: `NeedsPush` truth table (no upstream → true, ahead 0 → false, ahead 2 → true, detached → false, not repo → false) and `Reason.Text()` for every enum value

### Git runner and inspection

- [X] T010 Write `internal/gitx/runner.go`: `type Runner interface { Run(ctx, dir string, timeout time.Duration, args ...string) (stdout, stderr string, err error) }` and `ExecRunner{GitPath string, Verbose io.Writer}`; use `exec.CommandContext`, `-C dir`, and set the environment from contracts/git-commands.md: `GIT_TERMINAL_PROMPT=0`, `GIT_ASKPASS` to a no-op (`true`/`echo` on Unix, `cmd /c exit 1` on Windows via `runtime.GOOS`), `GIT_SSH_COMMAND=ssh -o BatchMode=yes` only when the user has not set it, `LC_ALL=C`, `GIT_OPTIONAL_LOCKS=0`; honour `PUSHALL_GIT` env override; when Verbose is non-nil write `$ git <args> → exit N` per call; return a typed `*ExitError{Code int, Stderr string, TimedOut bool, Cancelled bool}`
- [X] T011 [P] Write `internal/gitx/version.go`: `Available(ctx, r Runner) (version string, err error)` running `git --version`, parsing `git version X.Y[.Z]`, and returning an error when missing or below 2.20
- [X] T012 Write `internal/gitx/inspect.go` implementing each row of the contracts/git-commands.md "Inspection commands" table with a 15 s timeout: `TopLevel(ctx, path) (top string, isRepo bool)`, `CurrentBranch`, `Upstream`, `Remotes`, `AheadCount` (only when upstream exists), `Status` (porcelain v1 lines), `InProgressOp` (`rev-parse --git-path` for `MERGE_HEAD`, `rebase-merge`, `rebase-apply`, `CHERRY_PICK_HEAD` then `os.Stat`), `IdentityOK` (`config user.name`/`user.email`); plus `Inspect(ctx, path) (model.Candidate, error)` that fills a Candidate, marking `NestedRepo` when the cleaned toplevel differs from `path` (compare with `filepath.Clean` and case-insensitively on Windows/macOS)
- [X] T013 [P] Write `internal/gitx/errors.go`: `Classify(err error, stderr string) (model.Reason, detail string)` implementing the "Push error classification" table verbatim (case-insensitive substring, first hit wins; deadline → `Timeout`; cancelled → `Interrupted`; fallback `GitError` with last non-empty stderr line)
- [X] T014 [P] Unit-test `internal/gitx/errors_test.go` with one case per pattern row in contracts/git-commands.md plus the fallback, and `internal/gitx/version_test.go` for `git version 2.54.0 (Apple Git-157)`, `git version 2.19.1` (too old), and empty output
- [X] T015 Write a fake runner `internal/gitx/fake_runner.go` (`FakeRunner` keyed by joined args → scripted stdout/stderr/exit) and unit-test `Inspect` in `internal/gitx/inspect_test.go` for: clean repo with upstream ahead 2, detached HEAD, no upstream, no remotes, dirty with 3 porcelain lines, merge in progress, nested path, not a repo

### Integration-test harness (real git)

- [X] T016 Write `tests/integration/helpers_test.go`: `requireGit(t)`, `mkBareRemote(t, dir, name) string`, `mkRepo(t, dir, name, opts...)` that runs `git init`, sets local `user.name`/`user.email`, makes an initial commit, optionally adds `origin` pointing to a bare remote and pushes with `-u`; helpers `addUnpushedCommit`, `makeDirty`, `detach`, `startMerge` (two conflicting branches then `git merge` left unresolved), `remoteHead(t, bare) string`; all paths via `t.TempDir()` so they work on Windows

### UI primitives and report core

- [X] T017 Write `internal/ui/term.go`: `IsTTY(f *os.File) bool`, `NewStyle(noColor bool)` returning lipgloss styles or no-op styles, symbol set `✔ = – ✘ ○` with ASCII fallback `+ = - x o` when `noColor` or the console is not UTF-8 (on Windows check `GetConsoleOutputCP() == 65001` via `golang.org/x/sys/windows`, elsewhere check `LANG`/`LC_ALL` contain `UTF-8`); honour `NO_COLOR` env
- [X] T018 [P] Write `internal/ui/progress.go`: `Progress` with `Start(i, n int, name string)` printing `[i/N] name … ` without newline and `Done(result model.Result)` appending the outcome suffix (`pushed (2 commits)`, `up to date`, `not pushed: <reason text>`, `skipped`) and a newline; plain output when not a TTY
- [X] T019 Write `internal/report/report.go`: `Build(run model.Run) Report` grouping Results into the fixed Status order, computing `Counts`, counting `NotARepo` results separately as "folders" not "not pushed" for the summary, and `ExitCode(strict bool) int` per data-model.md: 0 all Pushed/UpToDate (SkippedByUser allowed unless strict), 1 any NotPushed or strict-skipped, 130 when `run.Interrupted`
- [X] T020 [P] Unit-test `internal/report/report_test.go`: grouping order, counts, and the four exit-code rules including `--strict` and interrupted

### Entry point skeleton

- [X] T021 Write `cmd/pushall/main.go` with `flag` parsing of every flag in contracts/cli.md (`--mode`, positional PATH, `--dirty`, `--message`, `--create-repo`, `--visibility`, `--no-review`, `--report`, `--timeout`, `--strict`, `--no-color`, `--verbose`, `--version`, `--help`) into `model.Options`, validating enum values (invalid → usage error exit 2), then calling `run.Run(ctx, opts, deps)` and `os.Exit` with its code; create `ctx` with `signal.NotifyContext(SIGINT, SIGTERM)` and exit immediately on a second signal
- [X] T022 [P] Unit-test flag parsing in `cmd/pushall/main_test.go`: defaults (`--dirty ask`, `--create-repo ask`, `--visibility private`, `--timeout 120`), each invalid enum value → exit 2, `--version` prints and exits 0

**Checkpoint**: `go build ./...` and `go test ./...` pass; no user-facing behaviour yet.

---

## Phase 3: User Story 1 - Push every project under a parent folder (Priority: P1) 🎯 MVP

**Goal**: Folder mode end-to-end: discover child repositories, push each current branch one at a time, handle uncommitted changes with a per-repo prompt (FR-007/007a), offer GitHub repository creation when no remote exists (FR-008/008a/008b), isolate failures (FR-009), show live progress (FR-010) and print the grouped text report with summary (FR-011/012/013).

**Independent Test**: quickstart.md Scenario 1 (five-folder fixture: alpha unpushed, beta dirty, gamma up to date, delta no remote, notes not a repo) produces the exact grouping described there and exit code 1; Scenario 3 exercises the dirty prompt; Scenario 5 exercises failure isolation.

### Tests for User Story 1

- [X] T023 [P] [US1] Unit-test discovery in `internal/discover/discover_test.go`: direct children only, directories only, dot-folders skipped, symlinked directory followed, case-insensitive sort (`Zeta` after `alpha`), root included first when it is a repo, empty folder → zero candidates
- [X] T024 [P] [US1] Integration test `tests/integration/push_test.go`: fixture of alpha (1 unpushed), gamma (up to date), notes (plain folder); run the orchestrator with `Dirty=PushOnly`, `CreateRepo=No`, `NoReview=true`; assert alpha `Pushed` and `remoteHead(alpha)` equals local HEAD, gamma `UpToDate`, notes `NotPushed/NotARepo`, processing order alpha→gamma→notes, exit code 0
- [X] T025 [P] [US1] Integration test `tests/integration/dirty_test.go`: beta dirty; with a fake prompter answering `CommitAndPush` assert a commit with the given message reaches the bare remote and `Result.Committed` is true; answering `PushOnly` → `UpToDate` with detail "uncommitted changes left untouched"; answering `Skip` → `SkippedByUser`; with `user.email` unset and `CommitAndPush` → `NotPushed/IdentityMissing` and no commit created
- [X] T026 [P] [US1] Integration test `tests/integration/noremote_test.go`: delta has no remote; with fake `github.Client` reporting not logged in → `NotPushed/NoRemoteLoginRequired`; logged in and prompter declines → `NotPushed/NoRemoteDeclined`; logged in and accepts → fake `CreateRepo` called with name `delta`, visibility `Private`, and result `Pushed` with `CreatedRemote` set; fake returns "already exists" error → prompter asked again for a name, second decline → `NotPushed/CreateRepoFailed`
- [X] T027 [P] [US1] Integration test `tests/integration/failures_test.go`: alpha with `origin` URL pointing to a non-existent path → `NotPushed` with `GitError` or `NetworkError` while the following repo is still processed; gamma detached → `NotPushed/DetachedHead`; repo with unresolved merge → `NotPushed/OperationInProgress`; repo with upstream but no `origin` remote name and no upstream set → `NotPushed/NoRemote`
- [X] T028 [P] [US1] Golden test `internal/report/render_text_test.go`: a fixed `model.Run` with one result per Status plus two `NotARepo` folders renders byte-for-byte to `internal/report/testdata/report.txt` in the layout of contracts/report-format.md "On-screen (text)" (ASCII symbols, no colour)

### Implementation for User Story 1

- [X] T029 [US1] Write `internal/discover/discover.go`: `Discover(ctx, root string, g gitx.Inspector) ([]model.Candidate, error)` per research R7: `os.ReadDir(root)`, keep `IsDir()` entries and symlinks that resolve to directories, skip names starting with `.`, sort with `strings.ToLower` comparison, call `g.Inspect` on each, and prepend the root itself when `Inspect(root).IsRepo`
- [X] T030 [US1] Write `internal/gitx/mutate.go`: `AddAll(ctx, path)` → `git add -A`; `Commit(ctx, path, msg)` → `git commit -m msg` (never `--no-verify`); `Push(ctx, path, timeout)` → `git push` when upstream exists, else `git push --set-upstream origin <branch>`; return `Classify`-ed reason on failure; add a compile-time guard test in `internal/gitx/forbidden_test.go` that greps the package source for `--force`, `reset`, `rebase`, `checkout`, `switch`, `clean`, `stash` and fails if found
- [X] T031 [P] [US1] Write `internal/github/gh.go`: `type Client interface { Available() bool; LoggedIn(ctx) bool; CreateRepo(ctx, path, name string, private bool, timeout) (url string, err error) }` and `GhClient` using `exec.LookPath("gh")`, `gh auth status --hostname github.com`, `gh -C path repo create NAME --private|--public --source . --remote origin --push`, then `gh -C path repo view --json url -q .url`; map stderr containing `already exists` to `ErrNameTaken`; reuse the gitx runner environment so `gh` never prompts; include `FakeClient` in `internal/github/fake.go`
- [X] T032 [US1] Write `internal/ui/prompts.go` with a `Prompter` interface (`Mode() model.Mode`, `Path(mode) string`, `Review([]model.Candidate) []model.Candidate`, `ConfirmStart(n int) bool`, `DirtyDecision(c model.Candidate) model.Decision`, `CreateRepoOffer(c model.Candidate) model.Decision`, `SaveReport(defaultPath string) (string, bool)`) and `HuhPrompter` implementing `DirtyDecision` (show `DirtyFiles`, select Commit/PushOnly/Skip, on Commit a text input prefilled `chore: bulk push on <YYYY-MM-DD HH:MM>`, then confirm "Apply this answer to all remaining repositories?") and `CreateRepoOffer` (confirm "Create GitHub repository `<name>` (private) and push?" with options to edit name and toggle visibility, then "Apply to all remaining repositories without a remote?") per contracts/cli.md steps 6; include `ScriptedPrompter` in `internal/ui/scripted.go` for tests
- [X] T033 [US1] Write `internal/ui/noninteractive.go`: `NonInteractivePrompter` that returns an error naming the flag that supplies each missing answer (`--mode`, `PATH`, `--dirty`, `--create-repo`, `--report`), used when stdin or stdout is not a TTY (exit 2 per contracts/cli.md "Non-interactive mode")
- [X] T034 [US1] Write `internal/run/run.go`: `Run(ctx, opts, deps{Git, GitHub, Prompter, Progress, Out io.Writer}) int` implementing the data-model.md state machine for folder mode: resolve mode/path (prompt when not in opts), validate path (exists, directory, readable; re-prompt on failure, exit 2 when non-interactive), `Discover`, print `Found N repositories under <root>`, confirm start, then for each candidate in order: `Progress.Start`; branch on `!IsRepo`/nested/detached/in-progress; if Dirty resolve decision from `opts.Dirty` or the remembered ApplyToAll decision or `Prompter.DirtyDecision` (check `IdentityOK` before committing); if no remotes resolve via `opts.CreateRepo`/remembered/`CreateRepoOffer` and `GitHub.LoggedIn`, retrying the name on `ErrNameTaken`; else `UpToDate` when upstream exists and ahead 0, else `Push`; record `Result` with `Duration`; on `ctx.Err()` mark the current result `Interrupted` and all remaining `NotProcessed`, set `run.Interrupted`; finally `report.Build`, render text to `Out`, return `ExitCode(opts.Strict)`
- [X] T035 [US1] Write `internal/report/render_text.go`: `RenderText(w io.Writer, rep Report, style ui.Style)` producing the contracts/report-format.md on-screen layout: header line `Push report — <date> — <root> — folder mode|single mode`, groups in fixed order omitting empty ones, `Not a Git repository: a, b` compact line, `Not processed (N)` only after interruption, name column capped at 40 chars with `…`, summary line `N pushed, N up to date, N not pushed, N skipped by user · N folders · <duration>`
- [X] T036 [US1] Wire `cmd/pushall/main.go`: check `gitx.Available` (exit 2 with message when missing/old), choose `HuhPrompter` or `NonInteractivePrompter` by `ui.IsTTY(os.Stdin) && ui.IsTTY(os.Stdout)`, construct `ExecRunner` (Verbose → `os.Stderr` when `--verbose`), `GhClient` (warn once on stderr when `gh` not installed), `Progress`, and call `run.Run`
- [X] T037 [US1] Run `go test ./...` and quickstart.md Scenarios 1, 3 and 5 manually on this machine; fix discrepancies until the report matches the contract exactly

**Checkpoint**: Folder mode is a complete MVP: `pushall --mode folder <dir>` discovers, prompts, pushes, isolates failures, and prints the report with the right exit code.

---

## Phase 4: User Story 2 - Push a single project (Priority: P2)

**Goal**: `--mode single <path>` (or the interactive "A single project" choice) pushes exactly one repository, rejects non-repositories with exit 2, and prints a one-entry report.

**Independent Test**: quickstart.md Scenario 2: gamma with a new commit → pushed, exit 0; `notes` → "not a Git repository", exit 2, no report.

### Tests for User Story 2

- [X] T038 [P] [US2] Integration test `tests/integration/single_test.go`: single mode on a repo with one unpushed commit → one `Pushed` result, remote head updated, exit 0; single mode on a plain folder → exit 2 and no report written to `Out`; single mode on a nested sub-directory of a repo → exit 2 with message naming the top-level path

### Implementation for User Story 2

- [X] T039 [US2] Extend `internal/run/run.go`: when `opts.Mode == SingleProject`, skip `Discover`, call `Git.Inspect(path)`, return exit 2 with `not a Git repository: <path>` when `!IsRepo`, or `path is inside repository <top>; run on <top> instead` when nested; build a one-candidate list and reuse the per-candidate loop; never show the review step in single mode
- [X] T040 [US2] Extend `internal/ui/prompts.go` `HuhPrompter.Mode()` with the two-option select ("A folder that contains my projects" / "A single project") and `Path(mode)` validation message `must be a Git repository` in single mode; header of the text report says `single mode`

**Checkpoint**: Both entry modes work independently.

---

## Phase 5: User Story 3 - Review the list before pushing (Priority: P3)

**Goal**: After discovery, show every candidate with branch and status, let the user uncheck entries, then confirm. Excluded repos become `SkippedByUser` "excluded in review" (FR-017). `--no-review` skips the step.

**Independent Test**: quickstart.md Scenario 4: uncheck alpha → alpha under "Skipped by user" with "excluded in review", nothing pushed for alpha, others processed.

### Tests for User Story 3

- [X] T041 [P] [US3] Unit-test `internal/ui/review_test.go`: the review row formatter renders `name  [branch]  ↑2`, `up to date`, `no upstream`, `no remote`, `dirty`, `detached`, `merge in progress` from the corresponding Candidate fields, and ASCII `^2` in no-color mode
- [X] T042 [P] [US3] Integration test `tests/integration/review_test.go`: three repos, `ScriptedPrompter.Review` excludes the first → it is `SkippedByUser` with detail `excluded in review`, the other two `Pushed`; with `NoReview=true` the prompter's `Review` is never called; declining `ConfirmStart` → exit 0 and a report with zero processed entries

### Implementation for User Story 3

- [X] T043 [US3] Write `internal/ui/review.go`: `FormatReviewRow(c model.Candidate, style) string` and `HuhPrompter.Review` as a multi-select with all candidates checked by default (non-repos and nested entries shown disabled/unchecked), returning the kept list; `HuhPrompter.ConfirmStart(n)` confirm "Push N repositories now?"
- [X] T044 [US3] Extend `internal/run/run.go`: after discovery and before the loop, unless `opts.NoReview` or single mode, call `Prompter.Review`, set `Excluded=true` on removed candidates (they stay in the result list as `SkippedByUser` with detail `excluded in review`), then `ConfirmStart`; on decline print an empty report and return 0

**Checkpoint**: Users can exclude repos before anything is pushed.

---

## Phase 6: User Story 4 - Keep the report (Priority: P3)

**Goal**: Render the report as Markdown per contracts/report-format.md and save it on request (`--report PATH`, `--report -` to disable, or the end-of-run prompt) with UTF-8 and `\n` line endings on all OSes (FR-016).

**Independent Test**: quickstart.md Scenario 5 with `--report ./report.md` writes a file whose entries match the screen; the golden Markdown test passes on all three CI runners (SC-005).

### Tests for User Story 4

- [X] T045 [P] [US4] Golden test `internal/report/render_markdown_test.go`: the same fixed `model.Run` used in T028 renders byte-for-byte to `internal/report/testdata/report.md` matching the contracts/report-format.md Markdown structure (header bullets, one table per group, `Not a Git repository` list), with `\n` endings regardless of `runtime.GOOS`
- [X] T046 [P] [US4] Integration test `tests/integration/report_test.go`: `ReportPath` set → file exists and contains every repository name exactly once; `ReportPath == "-"` → no file and prompter `SaveReport` not called; prompter accepts the default → file `push-report-YYYYMMDD-HHMMSS.md` created in the working directory

### Implementation for User Story 4

- [X] T047 [P] [US4] Write `internal/report/render_markdown.go`: `RenderMarkdown(w io.Writer, rep Report)` producing the contract layout: `# Push report`, bullets Date/Root/Mode/Tool/Duration/Summary, `## <Group> (N)` with `| Repository | Branch | Details |` (Pushed/UpToDate) or `| Repository | Branch | Reason |` (NotPushed/SkippedByUser/NotProcessed), `—` for empty branch, `Reason.Text()` plus `: <detail>` when present, timestamps `YYYY-MM-DD HH:MM:SS` local
- [X] T048 [P] [US4] Write `internal/report/save.go`: `DefaultReportPath(now time.Time) string` → `push-report-YYYYMMDD-HHMMSS.md`; `Save(path string, rep Report) error` writing UTF-8 with `\n` via `os.WriteFile` with mode `0644`, creating parent directories, returning a clear error on failure (reported to stderr, does not change the exit code)
- [X] T049 [US4] Extend `internal/run/run.go` and `internal/ui/prompts.go`: after printing the text report, if `opts.ReportPath == "-"` do nothing; if set, `Save` to it; otherwise (interactive) `Prompter.SaveReport(DefaultReportPath(now))` confirm "Save this report to `./push-report-…md`?" and save on yes; print `Report saved to <path>`

**Checkpoint**: All four user stories work independently.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Interrupt handling verification, documentation, release pipeline, and the quickstart pass.

- [X] T050 [P] Integration test `tests/integration/interrupt_test.go`: cancel the context while the second of three repos is pushing (use a pre-push hook that sleeps, or a `FakeRunner` that blocks until ctx done) → first `Pushed`, second `NotPushed/Interrupted`, third `NotProcessed`, `run.Interrupted` true, report still rendered, exit 130
- [X] T051 [P] Integration test `tests/integration/timeout_test.go`: `Timeout = 1s` with a pre-push hook that sleeps 5 s → `NotPushed/Timeout` with text `push exceeded 1 s`, next repo still processed
- [X] T052 [P] Write `README.md`: what the tool does, install via GitHub Releases download (Linux/macOS/Windows, including `chmod +x` and Windows SmartScreen note) and `go install github.com/aarjang/push-all-repo/cmd/pushall@latest`, prerequisites (`git` ≥ 2.20, optional `gh` login), the interactive flow, every flag from contracts/cli.md, exit codes, a sample report, PowerShell equivalent of the quickstart fixture, and the safety guarantees (no force-push, commits only on consent, private repos by default)
- [X] T053 [P] Write `.goreleaser.yaml`: builds for `linux`/`darwin`/`windows` × `amd64`/`arm64`, `main: ./cmd/pushall`, binary `pushall`, `-s -w -X main.version={{.Version}}` ldflags, archives `tar.gz` (zip on Windows) with README and LICENSE, checksums; and `.github/workflows/release.yml` running GoReleaser on `v*` tags with `GITHUB_TOKEN`
- [X] T054 [P] Add `version` variable in `cmd/pushall/main.go` (default `dev`, overridden by ldflags) printed by `--version` and placed in `Run.ToolVersion` for the report header
- [X] T055 Run `gofmt -l .`, `go vet ./...`, `go test -race ./...` from the repository root (`go.mod` directory) on this machine and fix all findings in the reported files
- [X] T056 Walk through every quickstart.md scenario (1–5, 7, 8 locally; 6 only if a GitHub login is available and the user agrees to create a real repository) and record the outcome of each in `specs/001-bulk-repo-push/quickstart.md` under a new "Validation log" heading with the date
- [ ] T057 Push the branch and confirm the CI matrix in `.github/workflows/ci.yml` is green on ubuntu, macos and windows; fix any OS-specific failure (paths, line endings, console code page)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies; T004–T006 parallel after T001–T003.
- **Foundational (Phase 2)**: depends on Phase 1; **blocks every user story**. Internal order: T007→T008→T009; T010→(T011, T012, T013)→(T014, T015); T016 independent; T017→T018; T019→T020; T021→T022.
- **US1 (Phase 3)**: depends on Phase 2. Tests T023–T028 first (they fail), then T029–T031 in parallel, T032→T033, then T034 (needs T029–T033), T035, T036, T037.
- **US2 (Phase 4)**: depends on Phase 2 and on `run.go` from US1 (T034). Can start once T034 exists.
- **US3 (Phase 5)**: depends on Phase 2 and T034; independent of US2 and US4.
- **US4 (Phase 6)**: depends on Phase 2 and T034 and the report core (T019); T047/T048 can be written in parallel with US2/US3.
- **Polish (Phase 7)**: T050/T051 need T034; T052–T054 can start any time after Phase 2; T055–T057 last.

### User Story Dependencies

- **US1**: none beyond Foundational. Delivers the MVP.
- **US2**: shares the per-candidate loop in `internal/run/run.go` with US1; adds a branch at the top of `Run`. Independently testable via Scenario 2.
- **US3**: adds one step between discovery and the loop; independently testable via Scenario 4.
- **US4**: adds rendering/saving after the loop; independently testable via Scenario 5 and the golden file.

### Parallel Opportunities

- Phase 1: T004, T005, T006 together.
- Phase 2: T009, T011, T013, T014, T016, T018, T020, T022 are all marked [P]; the chains T010→T012→T015 and T017, T019, T021 can proceed concurrently on different files.
- US1: all six test tasks T023–T028 together; then T029, T030, T031 together; T032 and T033 together.
- US2, US3 and US4 touch different files except `internal/run/run.go` and `internal/ui/prompts.go`; a single developer should do their `run.go` edits (T039, T044, T049) sequentially, while T047, T048, T041, T043 run in parallel.
- Phase 7: T050–T054 together.

---

## Parallel Example: User Story 1

```bash
# Write all US1 tests first (different files, all fail initially):
Task: "Unit-test discovery in internal/discover/discover_test.go"
Task: "Integration test tests/integration/push_test.go"
Task: "Integration test tests/integration/dirty_test.go"
Task: "Integration test tests/integration/noremote_test.go"
Task: "Integration test tests/integration/failures_test.go"
Task: "Golden test internal/report/render_text_test.go"

# Then the three independent implementation pieces:
Task: "Write internal/discover/discover.go"
Task: "Write internal/gitx/mutate.go"
Task: "Write internal/github/gh.go and fake.go"

# Then prompts, then the orchestrator that ties them together:
Task: "Write internal/ui/prompts.go and noninteractive.go"
Task: "Write internal/run/run.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup (T001–T006).
2. Phase 2 Foundational (T007–T022). Verify `go test ./...` passes.
3. Phase 3 US1 (T023–T037).
4. **STOP and VALIDATE** with quickstart Scenarios 1, 3, 5. This is a usable tool: folder mode with prompts, failure isolation, and the on-screen report.

### Incremental Delivery

1. MVP above → tag `v0.1.0` once T053 exists.
2. US2 single mode (T038–T040) → validate Scenario 2.
3. US3 review step (T041–T044) → validate Scenario 4.
4. US4 Markdown report (T045–T049) → validate Scenario 5 and golden parity.
5. Polish (T050–T057) → `v1.0.0`.

### Parallel Team Strategy

With two people: one does Phase 2 git/runner chain (T010–T015) while the other does model, UI, report and main (T007–T009, T017–T022). After US1, one takes US2+US3 (both edit `run.go` prompts), the other takes US4 and Polish docs/release (T047, T048, T052, T053).

---

## Notes

- [P] tasks touch different files and have no dependency on an unfinished task.
- Every integration test must pass on Windows: use `t.TempDir()`, `filepath.Join`, and never assume `/tmp` or `bash`.
- Never add `--force`, `--no-verify`, `reset`, `checkout` or `stash` to any git invocation; T030's guard test enforces this.
- Commit after each task or logical group; stop at any checkpoint to validate the story on its own.
