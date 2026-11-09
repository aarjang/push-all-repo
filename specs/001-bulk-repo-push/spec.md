# Feature Specification: Bulk Repo Push

**Feature Branch**: `001-bulk-repo-push`

**Created**: 2026-11-09

**Status**: Draft

**Input**: User description: "می‌خوام یه پروژه بنویسم که روی لینوکس و مک و ویندوز کار کنه. کلیت اینه که وقتی کاربر این پروژه رو اجرا کرد، یا یه مسیر پوشهٔ پروژه‌هاش رو بده یا مسیر یک پروژه رو، و این انتخاب ساده باشه. بعد ابزار تک‌به‌تک وارد پوشه‌ها و پروژه‌ها بشه و روی ریپو پوش کنه و در آخر گزارش بده که چی پوش شده و چی نشده. این نرم‌افزار رو می‌خوام روی گیت‌هاب پوش کنم."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Push every project under a parent folder (Priority: P1)

A developer keeps many Git projects side by side in one folder (for example `~/projects`). They run the tool, choose the "folder of projects" mode, give the folder path, and the tool walks through each child folder one by one. For each child that is a Git repository, it pushes the current branch to its remote. When the walk ends, the tool shows a report listing every project it found and whether the push succeeded, with a reason for each one that did not.

**Why this priority**: This is the core reason the tool exists. Without it there is no product; with only this story the tool already saves the user from opening and pushing dozens of repositories by hand.

**Independent Test**: Create a folder with four child Git repositories (one clean with unpushed commits, one already up to date, one with uncommitted changes, one with no remote configured) and a fifth plain folder. Run the tool on the parent folder. The first repo must be pushed, the second reported as already up to date, the third must trigger the commit/push/skip prompt, the fourth must trigger the create-GitHub-repository offer (and be reported as "no remote" if declined), and the plain folder reported as "not a Git repository".

**Acceptance Scenarios**:

1. **Given** a folder containing several Git repositories with unpushed commits, **When** the user selects "folder of projects" and provides that folder's path, **Then** the tool pushes each repository's current branch to its remote and the final report lists each one as pushed.
2. **Given** a child folder that is not a Git repository, **When** the walk reaches it, **Then** the tool skips it, continues with the next folder, and the report marks it as "not a Git repository".
3. **Given** a repository whose push fails (for example network error or rejected push), **When** the failure happens, **Then** the tool records the failure and its reason, continues with the remaining repositories, and the report lists it under "not pushed" with that reason.
4. **Given** a repository whose current branch is already fully pushed, **When** the walk reaches it, **Then** the report marks it as "already up to date" rather than as a failure.
5. **Given** a repository with uncommitted changes, **When** the walk reaches it, **Then** the tool shows the changed files and asks the user whether to commit and push, push only existing commits, or skip, and acts on the answer.
6. **Given** a repository with no remote and a logged-in GitHub account on the machine, **When** the walk reaches it, **Then** the tool offers to create a GitHub repository (default private, named after the folder); on acceptance it creates it, sets it as `origin`, pushes, and the report marks the repository as "pushed (new GitHub repository created)".

---

### User Story 2 - Push a single project (Priority: P2)

A developer wants to push just one project. They run the tool, choose the "single project" mode, give the project path, and the tool pushes that repository and shows a one-entry report.

**Why this priority**: The user explicitly asked for both entry modes. It is a thin variation of Story 1 that reuses the same push and report behaviour, so it is cheap to add but not sufficient on its own.

**Independent Test**: Run the tool in single-project mode on one repository with an unpushed commit; the commit must appear on the remote and the report must show that single repository as pushed.

**Acceptance Scenarios**:

1. **Given** a path to a Git repository with unpushed commits, **When** the user selects "single project" and provides that path, **Then** the tool pushes it and the report shows one entry marked as pushed.
2. **Given** a path that is not a Git repository, **When** the user selects "single project" with that path, **Then** the tool tells the user the folder is not a Git repository and does not attempt a push.

---

### User Story 3 - Review the list before pushing (Priority: P3)

Before any push happens, the tool shows the user the list of repositories it discovered, with each one's current branch and whether it has unpushed commits. The user can exclude any entries, then confirm to start pushing.

**Why this priority**: It protects the user from pushing a work-in-progress repository by accident, but the tool is still useful without it, so it comes after the two core modes.

**Independent Test**: Run the tool on a folder with three repositories, exclude one in the review step, confirm, and verify that only the other two are pushed and the excluded one appears in the report as "skipped by user".

**Acceptance Scenarios**:

1. **Given** the tool has discovered repositories under the chosen folder, **When** discovery finishes, **Then** the user sees each repository's name, current branch, and unpushed-commit status before any push is attempted.
2. **Given** the review list is shown, **When** the user excludes one repository and confirms, **Then** that repository is not pushed and the report marks it as "skipped by user".

---

### User Story 4 - Keep the report (Priority: P3)

After a run, the user can save the report to a file next to where they ran the tool so they can look at it later or share it.

**Why this priority**: Useful for users with many projects or recurring runs, but the on-screen report already satisfies the core need.

**Independent Test**: Complete a run, choose to save the report, and verify the saved file contains the same entries and statuses shown on screen.

**Acceptance Scenarios**:

1. **Given** a run has finished, **When** the user chooses to save the report, **Then** a report file is written containing every repository, its status, its reason (if not pushed), and the run date and time.

---

### Edge Cases

- The provided path does not exist, is a file rather than a folder, or cannot be read: the tool explains the problem and lets the user enter another path.
- The parent folder contains no Git repositories at all: the tool reports that nothing was found and ends without error.
- The parent folder is itself a Git repository (as well as containing others): the tool treats it as one of the projects to push.
- A repository is in a detached-HEAD state (no current branch): the tool skips it and reports "no current branch".
- A repository's current branch has no upstream branch on the remote: the tool pushes the branch to the default remote and sets the upstream, then reports it as pushed.
- A repository has more than one remote: the tool pushes to the branch's configured upstream remote; if none, to the remote named `origin`; if neither exists it falls back to the "no remote" flow (offer to create a GitHub repository, otherwise report "no remote").
- The user answers the uncommitted-changes prompt with "commit" but the repository has no committer name or email configured: the tool reports "cannot commit: Git identity not configured" and does not commit.
- Creating the GitHub repository succeeds but the first push to it fails: the tool reports the push failure and tells the user the remote was already created and set as `origin`, so the run is repeatable.
- Pushing requires credentials that are not available (no stored credentials, expired token): the tool does not hang waiting forever; it reports "authentication failed" for that repository and continues.
- A repository is in the middle of a merge, rebase, or has conflicts: the tool skips it and reports the state as the reason.
- A nested repository inside a project (submodule or vendored repo): it is not treated as a separate project; only direct children of the parent folder are considered.
- Very long paths, spaces, or non-Latin characters in folder names: handled on all three operating systems without failure.
- The user interrupts the run (for example with Ctrl+C): the tool stops after the current repository and still prints a report of what was done so far.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST run on Linux, macOS, and Windows with the same behaviour and the same report format.
- **FR-002**: On start, the tool MUST let the user pick one of two modes through a simple prompt: "a folder that contains my projects" or "a single project".
- **FR-003**: The tool MUST accept a path from the user for the chosen mode, validate that it exists and is readable, and ask again if it is not.
- **FR-004**: In folder mode, the tool MUST discover Git repositories among the direct child folders of the given folder (and the folder itself if it is a repository), without descending into nested repositories.
- **FR-005**: The tool MUST process repositories one at a time, in a stable and predictable order (alphabetical by folder name), so the user can follow progress.
- **FR-006**: For each repository, the tool MUST push the current branch to its upstream remote; if the branch has no upstream, it MUST push to the `origin` remote and set the upstream.
- **FR-007**: When a repository has uncommitted changes (modified, added, or deleted tracked files, or new untracked files that are not ignored), the tool MUST pause and ask the user, for that repository, whether to commit all changes and push, or to leave the changes untouched and push only existing commits, or to skip the repository entirely. If the user chooses to commit, the tool MUST show the list of files that will be committed and let the user accept a default commit message or type their own.
- **FR-007a**: The per-repository prompt in FR-007 MUST offer an "apply this answer to all remaining repositories" option so a run over many repositories does not require one answer per repository.
- **FR-008**: When a repository has no remote configured, the tool MUST offer to create a new GitHub repository for it, set that repository as the `origin` remote, and then push. The offer MUST show the proposed repository name (defaulting to the folder name) and visibility, and MUST let the user pick public or private (defaulting to private), change the name, or decline. If the user declines, the repository is reported as "not pushed: no remote".
- **FR-008a**: Creating a GitHub repository MUST use the user's existing GitHub login on that machine (for example a logged-in GitHub CLI); if no GitHub login is available, the tool MUST tell the user, skip creation, and report the repository as "not pushed: no remote (GitHub login required to create one)".
- **FR-008b**: If a GitHub repository with the proposed name already exists under the user's account, the tool MUST NOT overwrite or reuse it silently; it MUST tell the user and let them choose another name or decline.
- **FR-009**: A failure in one repository MUST NOT stop the run; the tool MUST record the failure and continue with the next repository.
- **FR-010**: The tool MUST show live progress while running, indicating which repository is being processed and the outcome of each one as it finishes.
- **FR-011**: At the end of a run, the tool MUST display a report that lists every folder considered, grouped into: pushed, already up to date, and not pushed, with a human-readable reason for every entry that was not pushed.
- **FR-012**: The report MUST include a summary count of pushed, already up to date, and not pushed entries.
- **FR-013**: The tool MUST exit with a status that distinguishes "everything pushed or up to date" from "at least one repository not pushed", so it can be used in scripts.
- **FR-014**: The tool MUST use the user's existing Git configuration, stored Git credentials, and existing GitHub login; it MUST NOT ask the user to type passwords or tokens into the tool itself.
- **FR-015**: The tool MUST never force-push, delete branches, rewrite history, or change the repository's current branch.
- **FR-016**: The tool MUST let the user save the final report to a file.
- **FR-017**: The tool MUST allow the user to review the discovered repositories and exclude any of them before pushing begins.
- **FR-018**: The tool MUST stop gracefully when the user interrupts it, finishing or abandoning only the repository in progress, and still show the report for completed entries.
- **FR-019**: The project MUST be publishable as an open-source repository on GitHub, including a README that explains installation and usage on all three operating systems, and a license file.

### Key Entities

- **Run**: One execution of the tool. Has a mode (folder of projects or single project), a root path, a start time, an end time, and a list of Repository Results.
- **Repository Candidate**: A folder examined during discovery. Has a path, a name, whether it is a Git repository, its current branch (if any), its upstream remote (if any), and whether it has unpushed commits or uncommitted changes.
- **Repository Result**: The outcome for one candidate. Has a status (pushed, pushed with new GitHub repository created, already up to date, not pushed, skipped by user) and a reason when not pushed (for example: not a Git repository, no remote, no current branch, authentication failed, push rejected, network error, repository in merge/rebase state, uncommitted changes left by user choice). Also records whether a commit was made by the tool and its message.
- **User Decision**: An answer given during the run to a per-repository prompt (commit / push only / skip; create GitHub repository yes / no, name, visibility), optionally remembered for the rest of the run.
- **Report**: The collection of Repository Results for a Run, with summary counts, shown on screen and optionally saved to a file.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user with no prior knowledge of the tool can start it, choose a mode, enter a path, and begin pushing in under 1 minute.
- **SC-002**: For a folder of 20 repositories that are all reachable, every repository with unpushed commits is pushed in a single run with no manual intervention.
- **SC-003**: 100% of repositories that could not be pushed appear in the final report with a reason a user can act on.
- **SC-004**: A single failing repository never prevents the remaining repositories from being processed in the same run.
- **SC-005**: The tool produces identical report content for the same set of repositories on Linux, macOS, and Windows.
- **SC-006**: A run over 20 repositories that are already up to date completes in under 2 minutes on a normal home internet connection.
- **SC-007**: The tool never rewrites repository history or switches branches; the only changes it makes to a repository are pushing commits, setting an upstream or `origin` remote when none existed, and creating a commit only after the user explicitly approved it for that repository.

## Assumptions

- Users already have Git installed and configured, with credentials for their remotes stored (SSH keys, credential helper, or a logged-in GitHub CLI). The tool reuses these rather than managing authentication.
- "Push on the repo" means pushing the current branch of each repository to its configured remote; pushing all branches or tags is out of scope for this version.
- Discovery looks only at direct children of the chosen folder (and the folder itself). Deep recursive scanning of nested folders is out of scope for this version.
- The tool is an interactive command-line program (text prompts and text report), not a graphical application. Command-line flags to skip the prompts are a nice-to-have, not required.
- The report is shown in English; localization into other languages is out of scope for this version.
- The report, when saved, is a plain text or Markdown file in the folder the user ran the tool from.
- Repositories may be hosted on GitHub or any other Git remote; pushing itself needs no GitHub-specific features. Only creating a new remote (FR-008) is GitHub-specific and depends on a GitHub login already present on the machine.
- When the tool commits on the user's behalf (FR-007), it stages all tracked changes and non-ignored new files; the default commit message is a short generic message including the date. Files matched by the repository's ignore rules are never committed.
- Newly created GitHub repositories default to private to avoid accidentally publishing code.
- The user runs the tool on their own machine against their own repositories; multi-user or server use is out of scope.
