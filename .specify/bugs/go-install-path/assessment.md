# Bug Assessment: `pushall` not found after `go install`

- **Slug**: go-install-path
- **Created**: 2026-11-09
- **Source**: pasted text (terminal transcript)
- **Verdict**: valid (documentation gap; the tool itself installed and built correctly)
- **Severity**: low

## Report (verbatim)

```text
~  go install github.com/aarjang/push-all-repo/cmd/pushall@latest        ok | 10:51:26
go: downloading github.com/aarjang/push-all-repo v0.0.0-20261109072047-32c2342c8830
~  pushall                                                           ok | 20s | 10:51:47
zsh: command not found: pushall
~  go install github.com/aarjang/push-all-repo/cmd/pushall@latest   127 err | 10:52:02
```

(The `127 err` on the last prompt is zsh reporting the exit status of the previous `pushall` command, not a failure of the second `go install`.)

## Symptom

`go install …@latest` completes successfully, but typing `pushall` afterwards gives `command not found`. The user expects the README's one-line Go install to leave a runnable command.

## Reproduction

1. On a machine where `$(go env GOPATH)/bin` (here `/Users/arjang/go/bin`) is **not** on `PATH`, run `go install github.com/aarjang/push-all-repo/cmd/pushall@latest`.
2. Run `pushall`.
3. Observe `command not found`.

Verified on the development machine during assessment:

- `@latest` resolved to pseudo-version `v0.0.0-20261109072047-32c2342c8830`, which is the merge commit of PR #1 on `main`, so the right code was fetched.
- The binary exists and is executable: `/Users/arjang/go/bin/pushall` (6.08 MB, built 10:52).
- `echo $PATH` contains no `/Users/arjang/go/bin` entry.

## Suspected Code Paths

- `README.md:25-29` — the "Or install with Go" section gives the `go install` command with no mention that the result lands in `$(go env GOPATH)/bin` and that this directory must be on `PATH`. This is the only place the project influences the outcome.
- No Go source is implicated; `cmd/pushall` built and installed exactly as designed.

## Root Cause Hypothesis

`go install` always writes binaries to `$GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. The Go toolchain installer does not add that directory to the shell `PATH`, and this user's zsh configuration does not include it. The README's install step omits this prerequisite, so a reader following it verbatim ends up with a correctly installed but unreachable binary. Confidence: **high** (the binary was located on disk and `PATH` was inspected).

## Proposed Remediation

**Preferred**: Extend the README "Or install with Go" section to state where the binary goes and how to make it reachable, with copy-paste lines for zsh/bash and PowerShell, e.g.

```bash
go install github.com/aarjang/push-all-repo/cmd/pushall@latest
# the binary lands in $(go env GOPATH)/bin; add it to PATH once:
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc && source ~/.zshrc
```

and for PowerShell: `$env:Path += ";$(go env GOPATH)\bin"` plus the permanent `setx` variant. Also add a one-line "If `pushall` is not found" troubleshooting note pointing at `$(go env GOPATH)/bin/pushall`.

**Alternatives**:
- Tell users to run `$(go env GOPATH)/bin/pushall` directly. Works, but hides the real fix and clutters every invocation.
- Publish the first GitHub Release (tag `v0.1.0`) so the "Download a release" path, which the README lists first, actually works; today no tag exists, so that section points at an empty Releases page. This is worth doing regardless but does not replace the PATH note, since `go install` users will still hit this.

**Files likely to change**:
- `README.md`

**Tests to add or update**:
- None in Go. Verification is manual: open a fresh shell after following the README, run `pushall --version`, expect `pushall dev` (or the tag version once released).

## Risks & Considerations

- Documentation-only change; no behavioural or API risk.
- The `--version` output for a `go install` build is `dev` because the version is only stamped by GoReleaser `-ldflags`. Consider reading `debug.ReadBuildInfo()` as a fallback so `go install` builds report the module pseudo-version; this is a separate small improvement, not part of this bug.
- The README's first install option (download from Releases) is currently a dead end until a `v*` tag is pushed.

## Open Questions

- None. The user's environment was inspected directly, so no clarification is needed.
