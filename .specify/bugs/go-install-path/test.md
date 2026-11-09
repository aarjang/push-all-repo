# Bug Verification: `pushall` not found after `go install`

- **Slug**: go-install-path
- **Tested**: 2026-11-09
- **Assessment**: ./assessment.md
- **Fix**: ./fix.md
- **Result**: verified

## Summary

The symptom still reproduces exactly as described when the README advice is not followed (clean shell, `~/go/bin` absent from `PATH` → exit 127), and each remedy the README now documents makes `pushall` resolvable and runnable. The regression suite is unaffected, as expected for a documentation-only change.

## Checks Performed

| Check | Command / Action | Result | Notes |
|-------|------------------|--------|-------|
| Reproduction (post-fix, remedy not applied) | `env -i HOME=<tmp> PATH=/usr/bin:/bin:<go> zsh -c 'pushall --version'` | pass | `command not found`, exit 127: confirms the diagnosis |
| zsh remedy from README | same clean env, run the README `echo … >> ~/.zshrc && source ~/.zshrc` line against a throwaway HOME, then `command -v pushall && pushall --version` | pass | resolved to `/Users/arjang/go/bin/pushall`, printed `pushall dev` |
| bash remedy from README | same with `~/.bashrc` under bash | pass | identical outcome |
| Full-path fallback from README | `$(go env GOPATH)/bin/pushall --version` in clean env | pass | `pushall dev` |
| PowerShell remedy from README | `[Environment]::SetEnvironmentVariable(...)` | not-run | `pwsh` is not installed on this machine |
| README structure | count of ``` fences | pass | 26 fences, all balanced |
| New / updated tests | n/a | skipped | fix added no tests (documentation change, per assessment) |
| Regression suite | `go vet ./...`, `go test ./...` | pass | all nine packages ok |
| Lint | `gofmt -l .` | pass | no files need formatting |
| Diff scope | `git diff --stat` | pass | only `README.md`, +28 lines |

## Output Excerpts

```text
=== symptom without remedy ===
zsh:1: command not found: pushall
exit=127

=== zsh snippet from README ===
/Users/arjang/go/bin/pushall
pushall dev
exit=0
```

## Residual Risks

- The PowerShell snippet was not executed here; it uses the documented `[Environment]::SetEnvironmentVariable` API with the `User` scope, which is standard, but it is unverified on a Windows machine.
- The fix only helps users who read the README. The user's own `~/.zshrc` was deliberately not modified during assessment, fix or test; they still need to run the documented line once.
- `pushall --version` from a `go install` build prints `dev`; a real version string is a separate follow-up noted in `fix.md`.
- The change is uncommitted on the already-merged `001-bulk-repo-push` branch and is not yet visible on GitHub.

## Recommendation

Close the bug: verified end-to-end for zsh and bash on macOS, with PowerShell documented but not executed. Commit the README change on a fresh branch from `main` and open a small PR; then consider the two follow-ups (first release tag, build-info version) separately.
