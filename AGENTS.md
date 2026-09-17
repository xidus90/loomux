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
  not held to the coverage rule. Moving to a newer upstream means redoing that
  patch.

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
- Commits carry the user as author and committer and credit no model or agent.
- Nobody but a human pushes.
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

- Gate: `.githooks/pre-commit` (gofmt, go vet, tests with coverage, pilot binary).
- `go run ./cmd/loomux dev covergate --profile coverage.out`
