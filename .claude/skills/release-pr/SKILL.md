---
name: release-pr
description: Use when opening a pull request against master in loomux, when updating one after new commits, or when a pull request's diff changed and its release label, changelog or commit grouping may no longer fit.
---

# Release PR

Every pull request to `master` releases through its label and body. The rules
(Conventional Commits, commit grouping, labels, changelog headings) are in `AGENTS.md` under "Rules"; read them there. This skill is the
order of work, the same for a new pull request and for an update.

## Steps

1. Read the state: `git diff master...HEAD` and `git log --oneline master..HEAD`.
   On an existing pull request also `gh pr view <n> --json labels,body,headRefOid`.
   If the current branch is `master`, stop: work moves to a branch first.
2. Group the commits by theme:
   - Save the head first: `git tag backup-<branch>-<short sha> HEAD`.
   - For each commit, ask whether it corrects something this branch
     introduced; it folds into that commit. It keeps its own commit only if
     the code it fixes was already on `master`:
     `git blame <merge-base> -- <file>` shows the line there.
   - Rebuild the branch from `git merge-base master HEAD`: per theme,
     `git checkout <old head> -- <paths>` and `git commit -F <file>`, so the
     gate runs on every commit. Each message is a Conventional Commit whose
     type says what the commit does for a user (`feat`, `fix`, ... ; `!` for a
     breaking change).
   - `git diff <old head> HEAD` must be empty afterwards.
3. Pick exactly one `release:*` label by the table in `AGENTS.md`. If every
   changed file is documentation, CI or a test, the label is `release:none`.
   Otherwise choose among major, minor and patch, and between two of them take
   the higher. The label is never lower than the highest commit type. Write the
   reason as one sentence.
4. Write the body to a file in the scratchpad, in exactly this shape:

   ```
   Release: <level> — <one-sentence reason>

   ## Summary
   - <what changes, from the user's point of view>

   ## Changelog
   ### <Added|Changed|Deprecated|Removed|Fixed|Security>
   - <entry>
   ```

   With `release:none` the `## Changelog` section is left out. The body ends
   after the last section: no attribution or "Generated with" line.
5. Check body, label and commits together:
   `git log --format=%B%x00 master..HEAD` split at the NUL bytes into a JSON
   array of messages in a scratch file, then
   `go run ./cmd/loomux dev release parse-body --labels release:<level> --body <file> --commits <json>`.
   Exit 1 prints the findings; fix and run again until exit 0.
6. Ask the human to push; agents never push, and the loomux guard refuses it.
   Give the exact command. First push: `git push -u origin <branch>`. After
   regrouping a pushed branch:
   `git push --force-with-lease=<branch>:<sha the remote had> origin <branch>`.
   Once they say it is pushed, `git ls-remote origin <branch>` must name
   `git rev-parse HEAD`; if not, say so and wait again.
7. New pull request: `gh pr create --base master --label release:<level> --title <title> --body-file <file>`.
   Existing one: `gh pr edit <n> --body-file <file>`, and swap the label with
   `--remove-label` / `--add-label` when the level changed. The title is the
   Conventional Commit header of the main change.
8. Delete the backup tag once the push landed, and report the link, the level
   and its reason.

## Changelog entries

- A change that has not been released yet is described as it will ship, not
  as a fix of itself: fixing a feature that is still unreleased adds no
  `Fixed` entry.
- Entries say what a user notices, not which files changed.
