# Bug Fix: `pushall` not found after `go install`

- **Slug**: go-install-path
- **Fixed**: 2026-11-09
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

The README's "Or install with Go" section now explains that `go install` places the binary in `$(go env GOPATH)/bin`, that this folder is not on `PATH` by default, and gives one-line fixes for zsh, bash and PowerShell plus a full-path fallback. No Go code changed; the tool installed correctly all along.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `README.md` | modified | Added PATH explanation, per-shell snippets, `pushall --version` check and full-path fallback under "Or install with Go" |

## Diff Highlights

```markdown
`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and the Go
installer does **not** add that folder to your `PATH`. If `pushall` says *command not found*
afterwards, add it once:

echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc && source ~/.zshrc
```

## Tests Added or Updated

- None. The assessment called for no Go tests; this is a documentation fix.

## Local Verification

- Commands run: in a clean `zsh -c` subshell with a minimal `PATH`, applied the documented line `export PATH="$PATH:$(go env GOPATH)/bin"` → `command -v pushall` printed `/Users/arjang/go/bin/pushall` and `pushall --version` printed `pushall dev`.
- Manual checks: re-read the rendered README section for correctness of the three snippets.

## Deviations from Assessment

None.

## Follow-ups

- Push a `v0.1.0` tag so the "Download a release" install path (listed first in the README) actually has assets; today the Releases page is empty.
- Consider reading `runtime/debug.ReadBuildInfo()` as a fallback for `--version`, so `go install` builds report the module pseudo-version instead of `dev`.
- This change is on the working tree only (branch `001-bulk-repo-push`, already merged); it still needs a commit, ideally on a fresh branch from `main`, and a push.
