# loomux

One Go binary for the hook path, the check chain and the knowledge system
that `ultraloom` and `ultra-brain` provided separately. Design:
`docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`.

## Where things live

- `cmd/loomux` is the entry point and nothing else; every command lives under `internal/`.
- Specs, plans and parity lists live under `docs/.superpowers/`.
- Recorded behaviour of the old tools lives under `testdata/cases/`.
- `third_party/toml` is `github.com/BurntSushi/toml` v1.6.0, pruned to what
  builds, with the local time zone resolved on first use instead of at start
  (see `internal/tz.go` there). `go.mod` replaces the module with it; it is
  not held to the coverage rule, and the gate neither formats nor vets it.
  Moving to a newer upstream means redoing that patch; verify it by copying
  upstream's `*_test.go` files and `internal/tag` into a scratch copy, calling
  the zone accessors there, and running `go test ./...` inside it.
- `internal/release` holds the release rules (next version, pull request
  body, changelog, cross build) behind `loomux dev release`. `ci/` holds the
  gate and the smoke test every forge runs. `.github/` holds only what is
  GitHub's: triggers, permissions, tokens and API calls. A second forge gets
  its own directory and calls the same scripts and subcommands.

## Languages

Everything that instructs an LLM, and everything that is not prose, is
English: this file, `CLAUDE.md`, `.claude/**`, code, comments, error
messages, commit messages. Documentation is multilingual: the project root maintains `README.md` (English, standard)
and `README.de.md` (German). Deep documentation under `docs/` is organized into
language subdirectories (`docs/en/`, `docs/de/`, etc.) with identical filenames across
languages to preserve clean link parity. Working papers under `docs/.superpowers/`
are German and never translated.

## Rules

- `.loomux/config.toml` is never written by an agent. It declares the areas a
  write barrier trusts and the policy that guards edits; propose changes, a
  human writes them.
- Coverage is 100% per function. A function may stay below only with
  `//coverage:exempt <reason>` on the line directly above `func`.
- No `init()` and no package-level variable parses embedded data; load on first use.
- Nobody works on `master`. Every change starts on a branch and reaches
  `master` only as a merged pull request. `.githooks/pre-commit` refuses a
  commit on `master` and `.githooks/pre-push` a push to it; the ruleset on
  GitHub is the barrier that holds regardless.
- Before a pull request is opened, and again before it is merged, its commits
  are grouped by theme: one commit per change. A later correction of something
  the same branch introduced (review fix, typo, follow-up) is folded into the
  commit that introduced it. Only the fix of a bug that already existed on
  `master` before the branch keeps a commit of its own. With an LLM the
  `release-pr` skill does this and the label below; without one, follow
  "Opening a pull request" in `README.md`.
- Commit messages follow Conventional Commits:
  `<type>[(<scope>)][!]: <description>`, optional body, optional footers.
  Types: `feat`, `fix`, `build`, `chore`, `ci`, `docs`, `style`, `refactor`,
  `perf`, `test`, `revert`. A breaking change carries `!` before the colon or
  a `BREAKING CHANGE:` footer. `.githooks/commit-msg` checks the header.
- The label of a pull request is never lower than its commits: a breaking
  change needs `release:major`, a `feat` at least `release:minor`, a `fix` at
  least `release:patch`. The label may be higher. `pr-label` checks it.
- Every pull request to `master` carries exactly one label `release:major`
  (breaking change to a command, flag, hook protocol, config format or exit
  code), `release:minor` (new feature, compatible), `release:patch` (bug fix
  or dependency update, compatible) or `release:none` (docs, CI or tests
  only). Whoever opens the pull request reads the diff, picks the label (the
  higher one when in doubt), sets it with `gh pr create --label`, and writes
  into the body a line `Release: <level> — <one-sentence reason>` (a
  convention; `parse-body` does not check it) and, unless
  `release:none`, a `## Changelog` block in Keep a Changelog form, English,
  from the user's point of view, headings only `Added`, `Changed`,
  `Deprecated`, `Removed`, `Fixed`, `Security`. Every non-blank line in that
  block is a `### ` heading or a `- ` entry; anything else is rejected. When
  the pull request changes, label and block follow. `loomux dev release
  parse-body` is the check.
- The one exception to the rules on commit authors and pushes: `chore(release): v*`
  commits and `v*` tags made by `.github/workflows/release.yml` through the
  `loomux-release` GitHub App. No agent uses that app.
- Commits carry the user as author and committer and credit no model or agent
  (see the release exception above).
- Nobody but a human pushes (see the release exception above). The loomux
  guard refuses a push by an agent; an agent names the exact command and
  waits.
- Performance measurements go chronologically into `docs/en/benchmarks.md` and
  `docs/de/benchmarks.md`: date and time, what was measured, baseline against
  change, cold and warm.
- Maintain the project READMEs (`README.md` and `README.de.md`) alongside
  implementation changes to reflect the actual state, capabilities, and roadmap.

## Commands

A fresh clone arms itself with two commands. Until the second one has run,
every hook in `.claude/settings.json` calls a binary that is not there and
does nothing:

```sh
git config core.hooksPath .githooks
go build -o bin/loomux.exe ./cmd/loomux
```

- Gate: `sh ci/gate.sh`; `.githooks/pre-commit` runs it, then rebuilds the pilot binary.
- The gate is `go run ./cmd/loomux check precommit`: the `[verify]` lanes of
  `.loomux/config.toml` over the presets. `check lint`, `check test` or
  `check coverage` run one kind; `check precommit --show` prints what runs.
