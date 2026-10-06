# Contributing to loomux

Bug reports, fixes and new flows are welcome. Report a vulnerability privately
as described in [`SECURITY.md`](SECURITY.md), never in a public issue.

## Before you start

For anything larger than a small fix, open an issue first and describe what
you want to change. loomux is maintained by one person; agreeing on the shape
before the code saves both sides a rewrite.

## Setting up a clone

You need Go (the version in `go.mod`; the toolchain line fetches it) and Git.
On a fresh clone, run once:

```sh
go run ./cmd/loomux init --yes
```

It points `core.hooksPath` at `.githooks` and builds `bin/loomux.exe`. From
then on the `pre-commit` hook runs the gate, refuses a commit on `master`, and
`commit-msg` checks the commit header. Run the gate by hand with
`sh ci/gate.sh`.

## Rules the gate and review hold you to

The full list is under "Rules" in [`AGENTS.md`](AGENTS.md). The ones a first
contribution meets most often:

- Work on a branch; nobody commits to `master`.
- One commit per change, as a Conventional Commit (`fix(cli): …`,
  `feat: …`, `docs: …`). A correction of something your branch introduced is
  folded into that commit. The message says what changed in terms of the
  code.
- Coverage is 100 % per function. A function may stay below only with
  `//coverage:exempt <reason>` on the line directly above `func`.
- No `init()`, and no package-level variable that parses embedded data.
- Code, comments, error messages and commit messages are English.
  Documentation exists in English and German: `README.md` and
  `README.de.md`, and `docs/en/` and `docs/de/` with the same file names. A
  change to one language changes the other in the same pull request.
- A new flow is data only; see
  [Contributing a Flow](docs/en/flows.md#5-contributing-a-flow).

## Opening a pull request

Follow [Opening a pull request](README.md#opening-a-pull-request) in the
README: group the commits, write the body with its `Release:` line and, unless
the change is docs, CI or tests only, a `## Changelog` block, and check it
with `go run ./cmd/loomux dev release parse-body`.

From a fork you cannot set labels on this repository. Open the pull request
without one and name the level you think fits in the `Release:` line; the
maintainer sets the label. Until then the `pr-label` check fails. CI for a
fork's pull request may wait until the maintainer approves the run.

## Licence

loomux is published under the PolyForm Noncommercial License 1.0.0
([`LICENSE.md`](LICENSE.md)). By submitting a contribution you agree that it
is published under the same licence.
