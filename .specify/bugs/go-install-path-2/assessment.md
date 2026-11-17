# Bug Assessment: `pushall` still not found after re-running `go install`

- **Slug**: go-install-path-2
- **Created**: 2026-11-09
- **Source**: pasted text (terminal transcript)
- **Verdict**: invalid (duplicate of `go-install-path`; environment action on the user's machine still pending, no project defect)
- **Severity**: low

## Report (verbatim)

```text
~  go install github.com/aarjang/push-all-repo/cmd/pushall@latest   127 err | 12:32:15
~  pushall                                                      ok | 4s | 13:18:15
zsh: command not found: pushall
~  go install github.com/aarjang/push-all-repo/cmd/pushall@latest   127 err | 13:18:19
```

(The `127 err` labels are zsh showing the exit status of the *previous* command, i.e. the failed `pushall` lookups. Both `go install` runs succeeded.)

## Symptom

Identical to `go-install-path`: `go install` succeeds and rebuilds the binary, but `pushall` is not found in the shell.

## Reproduction

Verified again on the user's machine during this assessment:

1. `/Users/arjang/go/bin/pushall` exists, rebuilt at 13:18 (6.08 MB), so the second `go install` worked.
2. `GOBIN` is unset, so `go install` targets `$(go env GOPATH)/bin` = `/Users/arjang/go/bin`.
3. A login zsh's `PATH` contains `/Users/arjang/.local/go-toolchain/go/bin` (the Go compiler's own directory, added by `~/.zshrc:126`) but **not** `/Users/arjang/go/bin`.
4. Therefore `pushall` cannot be resolved by name.

## Suspected Code Paths

- None in this repository. `README.md` already documents the fix (merged in PR #2, commit `f50cb0e`, now on `main` at `a4e718a`).
- `~/.zshrc:126` on the user's machine: exports the toolchain `bin`, which is a different directory from `GOPATH/bin`. This is the only place that needs to change, and it is outside the project.

## Root Cause Hypothesis

Same as `go-install-path`: `GOPATH/bin` is not on `PATH`. The README remedy was merged but has not yet been applied to this shell. Confidence: **high** (binary located, `PATH` and rc files inspected directly).

Side note, not a cause: `go list -m github.com/aarjang/push-all-repo@latest` still resolves to the PR #1 merge (`v0.0.0-20261109072047-32c2342c8830`) because the Go module proxy caches `@latest` for a while. The newer `main` differs only in documentation, so the installed binary is functionally current.

## Proposed Remediation

**Preferred**: no change to the project. On the user's machine, run the line the README now gives, once:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc && source ~/.zshrc
```

then `pushall --version` → `pushall dev`.

**Alternatives**:
- Set `GOBIN` to a directory already on `PATH` (for example `export GOBIN="$HOME/.local/bin"` if that is on `PATH`) and re-run `go install`. Works, but moves every future Go tool as well.
- Run by full path: `~/go/bin/pushall`.

**Files likely to change**:
- `~/.zshrc` (user's machine, outside the repository)

**Tests to add or update**:
- None.

## Risks & Considerations

- Editing `~/.zshrc` is a change to the user's shell configuration and should only be done with their explicit go-ahead.
- Once a `v0.1.0` tag exists, the Releases download path offers a way to install without touching `PATH` semantics of `go install` at all (the user chooses where to put the binary).

## Open Questions

- None.
