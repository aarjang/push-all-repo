# Contract: Command-Line Interface

**Binary**: `pushall` | **Feature**: 001-bulk-repo-push

## Invocation

```text
pushall [flags] [PATH]
```

Flags may appear before or after `PATH`. With no flags and no PATH the tool is fully interactive (FR-002, FR-003). Flags pre-answer prompts; a prompt is shown only when its answer is not already supplied.

## Flags

| Flag | Type | Default | Prompt it replaces |
|------|------|---------|--------------------|
| `--mode folder\|single` | string | prompt | Mode selection (FR-002) |
| `PATH` (positional) | string | prompt | Root path (FR-003) |
| `--dirty commit\|push-only\|skip\|ask` | string | `ask` | Uncommitted-changes decision for all repos (FR-007/FR-007a) |
| `--message TEXT` | string | `chore: bulk push on <date time>` | Commit message when `--dirty commit` |
| `--create-repo yes\|no\|ask` | string | `ask` | GitHub repo creation offer for all repos without remote (FR-008) |
| `--visibility private\|public` | string | `private` | Visibility for created repos |
| `--no-review` | bool | false | Skips the review/exclude step (FR-017) |
| `--report PATH` | string | prompt | Save Markdown report to PATH; `--report -` disables saving and the prompt |
| `--timeout SECONDS` | int | 120 | Per-push timeout |
| `--strict` | bool | false | User-skipped repos count as failures for the exit code |
| `--no-color` | bool | auto | Plain output; also implied when stdout is not a TTY |
| `--verbose` | bool | false | Echo every `git`/`gh` command and its exit status to stderr |
| `--version` | bool | | Print version and exit 0 |
| `--help` | bool | | Print usage and exit 0 |

Environment: `NO_COLOR` (any value) behaves like `--no-color`. `PUSHALL_GIT` overrides the `git` executable path (used by tests).

## Interactive flow

1. **Mode** (select): "A folder that contains my projects" / "A single project".
2. **Path** (text input, validated): must exist, be a directory, be readable; in single mode must be a Git repository top-level. On failure the error is shown and the input is repeated.
3. **Discovery** (no prompt): prints `Found N repositories under <root>` (folder mode).
4. **Review** (multi-select, all checked by default; skipped with `--no-review` or in single mode): one row per repository showing `name  [branch]  ↑N | up to date | no upstream | no remote | dirty | detached | merge in progress`. Unchecking excludes the repo.
5. **Confirm** (confirm): "Push N repositories now?" No → exit 0 with an empty report.
6. **Per-repository loop**, for each in order:
   - Progress line: `[i/N] name … ` then outcome suffix.
   - If dirty and `--dirty ask`: show changed files, then select: "Commit all changes and push" / "Push existing commits only" / "Skip this repository"; followed by a confirm "Apply this answer to all remaining repositories?". For commit: text input with default message.
   - If no remote and `--create-repo ask` and `gh` is logged in: confirm "Create GitHub repository `<name>` (private) and push?" with options to change name and visibility; then "Apply to all remaining repositories without a remote?".
7. **Report** printed to stdout (see [report-format.md](report-format.md)).
8. **Save** (confirm, unless `--report` given): "Save this report to `./push-report-<timestamp>.md`?".

## Output streams

- **stdout**: prompts (when TTY), progress lines, final report.
- **stderr**: usage errors, `--verbose` command echo, warnings (e.g. `gh` not installed).

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | All repositories pushed or already up to date (user-skipped allowed unless `--strict`) |
| 1 | At least one repository not pushed |
| 2 | Usage error, unreadable path, single path not a repo, or `git` not found |
| 130 | Interrupted by user; partial report was printed |

## Non-interactive mode

When stdin or stdout is not a TTY:
- The mode and path prompts cannot be shown, so a missing `--mode` or `PATH` is a usage error (exit 2) naming the flag.
- The review step is skipped and the run starts without confirmation.
- A per-repository prompt that would be needed (`--dirty ask`, `--create-repo ask`) marks that repository as not pushed with reason "answer required (no terminal): pass --dirty …" and prints the message once to stderr; the other repositories are still processed and the exit code is 1.
- The save-report prompt is skipped; pass `--report PATH` to save a file.

## Examples

```bash
pushall
pushall --mode folder ~/projects
pushall --mode single ~/projects/my-app --dirty push-only --report -
pushall --mode folder ~/projects --no-review --dirty commit --create-repo yes --visibility private --report ./today.md
```
