# pushall

Push every Git repository under a folder, one at a time, and get a report of what was pushed and what was not.

You point `pushall` at either **a folder that contains your projects** or **a single project**. It walks the repositories in order, pushes each current branch to its remote, asks you what to do when a repository has uncommitted changes, offers to create a GitHub repository when one has no remote, and ends with a grouped report you can save as Markdown.

Works the same on **Linux, macOS and Windows**. It is a single binary with no runtime to install.

## Install

### Download a release

Grab the archive for your OS from the [Releases page](https://github.com/aarjang/push-all-repo/releases), unpack it, and put `pushall` (or `pushall.exe`) somewhere on your `PATH`.

Linux / macOS:

```bash
tar -xzf pushall_*_linux_amd64.tar.gz && chmod +x pushall && sudo mv pushall /usr/local/bin/
```

On macOS the first run may be blocked by Gatekeeper; allow it under *System Settings → Privacy & Security*, or run `xattr -d com.apple.quarantine pushall`.

Windows: unzip, move `pushall.exe` to a folder on your `PATH`. SmartScreen may warn about an unknown publisher the first time; choose *More info → Run anyway*.

### Or install with Go

```bash
go install github.com/aarjang/push-all-repo/cmd/pushall@latest
```

## Prerequisites

- **Git 2.20 or newer** on your `PATH`, with `user.name` and `user.email` configured if you want `pushall` to commit for you.
- Your usual Git credentials already working (SSH keys or a credential helper). `pushall` never asks for passwords or tokens; it reuses whatever `git push` would use.
- **Optional:** the [GitHub CLI](https://cli.github.com/) logged in (`gh auth login`). It is only needed to create a GitHub repository for a project that has no remote yet.

## Usage

Fully interactive:

```bash
pushall
```

1. Choose *A folder that contains my projects* or *A single project*.
2. Enter the path.
3. Review the list of repositories found (branch, how many commits are unpushed, dirty, no remote…) and untick any you do not want to push.
4. Confirm. Repositories are processed one at a time with a live progress line.
5. For a repository with **uncommitted changes** you are asked: commit everything and push, push existing commits only, or skip. You can apply the same answer to the rest of the run.
6. For a repository with **no remote** you are offered to create a GitHub repository (private by default), which is then set as `origin` and pushed.
7. The report is printed, and you can save it as Markdown.

Scripted, no prompts:

```bash
pushall --mode folder ~/projects --no-review --dirty push-only --create-repo no --report ./today.md
```

### Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--mode folder\|single` | prompt | What `PATH` is |
| `PATH` | prompt | Folder of projects, or one repository |
| `--dirty ask\|commit\|push-only\|skip` | `ask` | What to do with uncommitted changes |
| `--message TEXT` | `chore: bulk push on <date>` | Commit message when committing |
| `--create-repo ask\|yes\|no` | `ask` | Offer to create a GitHub repo when there is no remote |
| `--visibility private\|public` | `private` | Visibility of created repositories |
| `--no-review` | off | Skip the review/untick step |
| `--report PATH` | prompt | Save the Markdown report to `PATH`; `-` never saves |
| `--timeout SECONDS` | `120` | Per-push timeout |
| `--strict` | off | Repositories you skipped count as failures |
| `--no-color` | auto | Plain output (also when `NO_COLOR` is set or output is not a terminal) |
| `--verbose` | off | Echo every `git` / `gh` command to stderr |
| `--version`, `--help` | | |

When no terminal is attached, every prompt must be answered by a flag; `pushall` tells you which one is missing and exits with code 2.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Everything pushed or already up to date |
| `1` | At least one repository was not pushed |
| `2` | Bad arguments, unreadable path, or `git` missing |
| `130` | Interrupted with Ctrl+C (a partial report is still printed) |

## What the report looks like

```text
Push report — 2026-11-09 14:03:17 — /Users/me/projects — folder mode

Pushed (3)
  ✔ api-server   main     2 commits pushed
  ✔ website      develop  1 commit pushed; committed by tool: "chore: bulk push on 2026-11-09 14:03"
  ✔ new-tool     main     pushed to new GitHub repository https://github.com/me/new-tool (private)

Already up to date (2)
  = dotfiles     main
  = notes        main     (uncommitted changes left untouched)

Not pushed (2)
  ✘ legacy-app   main     authentication failed
  ✘ scratch      -        no current branch (detached HEAD)

Skipped by user (1)
  – experiments  main     excluded in review

Not a Git repository: Downloads, screenshots

Summary: 3 pushed, 2 up to date, 2 not pushed, 1 skipped by user · 8 folders · 41 s
```

The saved Markdown file has the same content as tables, one per group.

## What pushall will never do

- It never force-pushes, deletes branches, rewrites history, switches branches or touches your stash.
- It only commits after you said yes for that repository (or passed `--dirty commit`), and it shows you the files first.
- It never asks for or stores passwords or tokens.
- New GitHub repositories are private unless you choose otherwise.
- It only looks at the direct children of the folder you give it; it does not scan recursively.

## How it works

`pushall` shells out to your installed `git` for every operation, so your SSH config, credential helpers and hooks all apply. Prompts are disabled for those calls (`GIT_TERMINAL_PROMPT=0`, SSH batch mode) so a missing credential fails fast with a clear reason instead of hanging. Creating repositories uses `gh repo create`.

## Try it on a throwaway fixture

Linux / macOS (bash):

```bash
mkdir -p /tmp/pa/remotes /tmp/pa/projects && cd /tmp/pa/projects && \
for r in alpha beta gamma; do git init -q --bare -b main ../remotes/$r.git; git init -q -b main $r && (cd $r && git commit -q --allow-empty -m init && git remote add origin ../../remotes/$r.git && git push -q -u origin main); done && \
(cd alpha && git commit -q --allow-empty -m "unpushed") && (cd beta && echo x > dirty.txt) && \
git init -q -b main delta && (cd delta && git commit -q --allow-empty -m init) && mkdir notes
```

Windows (PowerShell):

```powershell
New-Item -ItemType Directory -Force C:\pa\remotes, C:\pa\projects | Out-Null; Set-Location C:\pa\projects
foreach ($r in 'alpha','beta','gamma') { git init -q --bare -b main ..\remotes\$r.git; git init -q -b main $r; Push-Location $r; git commit -q --allow-empty -m init; git remote add origin ..\..\remotes\$r.git; git push -q -u origin main; Pop-Location }
Push-Location alpha; git commit -q --allow-empty -m unpushed; Pop-Location
Set-Content beta\dirty.txt x
git init -q -b main delta; Push-Location delta; git commit -q --allow-empty -m init; Pop-Location
New-Item -ItemType Directory notes | Out-Null
```

Then:

```bash
pushall --mode folder /tmp/pa/projects --no-review --dirty push-only --create-repo no --report -
```

## Development

```bash
make test
```

Tests use real `git` against temporary repositories with local bare remotes, so they need no network. CI runs them on Ubuntu, macOS and Windows.

## License

MIT. See [LICENSE](LICENSE).
