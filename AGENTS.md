# loomux

One Go binary for the hook path, the check chain and the knowledge system
that `ultraloom` and `ultra-brain` provided separately. Design:
`docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`.

## Where things live

- `cmd/loomux` is the entry point and nothing else; every command lives under `internal/`.
- Specs, plans and parity lists live under `docs/.superpowers/`.
- Recorded behaviour of the old tools lives under `testdata/cases/`.

## Languages

Everything that instructs an LLM, and everything that is not prose, is
English: this file, `CLAUDE.md`, `.claude/**`, code, comments, error
messages, commit messages. Documentation is bilingual: `X.md` is English and
the standard, `X.de.md` sits beside it. Working papers under
`docs/.superpowers/` are German and never translated.

## Rules

- `.loomux/config.toml` is never written by an agent. It declares the areas a
  write barrier trusts and the policy that guards edits; propose changes, a
  human writes them.
- Coverage is 100% per function. A function may stay below only with
  `//coverage:exempt <reason>` on the line directly above `func`.
- No `init()` and no package-level variable parses embedded data; load on first use.
- Commits carry the user as author and committer and credit no model or agent.
- Nobody but a human pushes.
- Performance measurements go chronologically into `docs/benchmarks.md` and
  `docs/benchmarks.de.md`: date and time, what was measured, baseline against
  change, cold and warm.
- Maintain the project READMEs (`README.md` and `README.de.md`) alongside
  implementation changes to reflect the actual state, capabilities, and roadmap.

## Commands

- Gate: `.githooks/pre-commit` (gofmt, go vet, tests with coverage, pilot binary).
- `go run ./cmd/loomux dev covergate --profile coverage.out`
