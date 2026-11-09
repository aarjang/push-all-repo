# Implementation Plan: Bulk Repo Push

**Branch**: `001-bulk-repo-push` | **Date**: 2026-11-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-bulk-repo-push/spec.md`

## Summary

An interactive, cross-platform command-line tool (`pushall`) that takes either a folder of projects or a single project path, walks the Git repositories one at a time, pushes each current branch to its remote, asks the user what to do about uncommitted changes, offers to create a GitHub repository when a project has no remote, and ends with a grouped report of what was pushed and what was not, optionally saved as Markdown. Built in Go as a single static binary; all Git and GitHub operations are delegated to the user's installed `git` and `gh` executables so their existing credentials are reused. Full decision log in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.24+

**Primary Dependencies**: standard library (`os/exec`, `flag`, `context`, `os/signal`); `github.com/charmbracelet/huh` (prompts); `github.com/charmbracelet/lipgloss` (terminal styling); `github.com/stretchr/testify` (tests only). External executables at runtime: `git` (required, ≥ 2.20), `gh` (optional, only for creating GitHub repositories).

**Storage**: N/A for state. One optional output file: the Markdown report written to the current working directory on request.

**Testing**: `go test ./...` with unit tests and offline integration tests that use temporary repositories and bare `file://` remotes; GitHub Actions matrix on Linux, macOS, Windows.

**Target Platform**: Linux, macOS, Windows 10+ (amd64 and arm64), terminal application.

**Project Type**: Single CLI project.

**Performance Goals**: Discovery of 50 child folders in under 2 s; a run over 20 up-to-date repositories completes in under 2 minutes on a home connection (SC-006); prompt response feels instant (< 100 ms to render).

**Constraints**: No network access except through `git`/`gh`; never prompts for secrets itself; never force-pushes, rewrites history, or switches branches (FR-015); must not hang on credential prompts (non-interactive Git env + per-push timeout, default 120 s); graceful Ctrl+C with partial report (FR-018); identical report content across OSes (SC-005).

**Scale/Scope**: One user, one machine, typically 1–100 repositories per run; ~6 internal packages, ~2–3k lines of Go including tests.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

`.specify/memory/constitution.md` is still the unfilled Spec Kit template (no ratified principles, version or dates). There are therefore **no project-specific gates** to evaluate. In their place this plan is checked against the Spec Kit default principles the template suggests, and the result is recorded so a future constitution can confirm or override it:

| Principle (template default) | Status | Evidence |
|------------------------------|--------|----------|
| Simplicity / YAGNI | PASS | Single project, standard-library-first, two small UI libraries, no database, no plugin system, no config file in v1. |
| CLI interface, text in/out | PASS | Interactive prompts plus optional flags; report to stdout and Markdown file; exit codes defined ([contracts/cli.md](contracts/cli.md)). |
| Test-first / testability | PASS | Git and GitHub wrappers sit behind interfaces; offline integration tests with bare remotes; CI matrix on all three OSes. |
| Observability | PASS | Live per-repository progress lines; `--verbose` echoes every external command; every non-pushed entry carries a reason. |
| Safety of user data | PASS | No force-push, no history rewrite, no branch switch; commits only after explicit per-repo consent; new GitHub repos private by default. |

**Recommendation**: run `/speckit-constitution` to ratify these (or different) principles before `/speckit-implement`. No violations to justify; Complexity Tracking is empty.

**Post-design re-check (after Phase 1)**: PASS. The data model adds no persistence, the contracts expose one binary with eight flags, and the package layout below has no layer without a clear owner.

## Project Structure

### Documentation (this feature)

```text
specs/001-bulk-repo-push/
├── plan.md              # This file
├── research.md          # Phase 0: decisions and alternatives
├── data-model.md        # Phase 1: entities, states, validation
├── quickstart.md        # Phase 1: end-to-end validation scenarios
├── contracts/
│   ├── cli.md           # Command, flags, prompts, exit codes
│   ├── report-format.md # On-screen and Markdown report contract
│   └── git-commands.md  # Exact git/gh invocations and how output is interpreted
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
cmd/
└── pushall/
    └── main.go              # Entry point: flags → run.Run → exit code

internal/
├── discover/                # Find candidate repositories under a root
│   ├── discover.go          # ReadDir, filter, sort, include root
│   └── discover_test.go
├── gitx/                    # Thin wrapper over the git executable
│   ├── runner.go            # exec with context, env hardening, timeout, verbose echo
│   ├── inspect.go           # toplevel, branch, upstream, remotes, ahead count, dirty, in-progress
│   ├── mutate.go            # add -A, commit, push, push --set-upstream
│   ├── errors.go            # classify stderr → reason (auth, rejected, network, …)
│   └── gitx_test.go
├── github/                  # Thin wrapper over the gh executable
│   ├── gh.go                # Available(), LoggedIn(), CreateRepo()
│   └── gh_test.go           # uses a fake runner
├── model/                   # Shared types: Run, Candidate, Result, Status, Reason, Decision
│   └── model.go
├── ui/                      # All user interaction (huh/lipgloss), TTY detection, plain fallback
│   ├── prompts.go           # mode, path, review/exclude, dirty decision, create-repo offer, save report
│   ├── progress.go          # per-repo progress lines
│   └── ui_test.go
├── report/                  # Build and render the report
│   ├── report.go            # grouping, summary counts, exit-code mapping
│   ├── render_text.go
│   ├── render_markdown.go
│   ├── save.go
│   └── report_test.go
└── run/                     # Orchestrator: the one-repo-at-a-time loop
    ├── run.go
    └── run_test.go

tests/
└── integration/             # Real git against temp repos + bare file:// remotes
    ├── helpers_test.go      # mkRepo, mkBareRemote, commit, etc.
    ├── push_test.go
    ├── dirty_test.go
    ├── noremote_test.go
    └── interrupt_test.go

.github/
└── workflows/
    ├── ci.yml               # vet + test matrix (ubuntu, macos, windows)
    └── release.yml          # GoReleaser on tag

.goreleaser.yaml
go.mod
go.sum
README.md
LICENSE                      # MIT
```

**Structure Decision**: Single Go module, one binary under `cmd/pushall`, logic split into small `internal/` packages by responsibility. `gitx` and `github` expose interfaces (`gitx.Git`, `github.Client`) so `run` and `ui` are unit-testable with fakes, while `tests/integration` exercises the real `git` binary on every OS in CI. Nothing is exported outside `internal/` because the project is an application, not a library.

## Complexity Tracking

No constitution violations; nothing to justify.
