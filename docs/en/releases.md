# Releases and pull requests

[Overview](README.md)

This page covers how a merged pull request becomes a release, how a beta is cut by hand, how to open a pull request, and how maintainers set the release machinery up.

## How a pull request becomes a release

Every merged pull request to `master` is released according to its label:

| Label | Meaning | Version |
|---|---|---|
| `release:major` | Breaking change to a command, flag, hook protocol, config format or exit code | `X+1.0.0` |
| `release:minor` | New feature, compatible | `X.Y+1.0` |
| `release:patch` | Bug fix or dependency update, compatible | `X.Y.Z+1` |
| `release:none` | Docs, CI or tests only | no release |

A release without a suffix is stable; one with a suffix (`1.1.0-beta.1`) is a
pre-release. What changed is in [`CHANGELOG.md`](../../CHANGELOG.md); an entry is
written only with a stable release, and the entries marked `-beta` are the
count before 1.0.0.

The compatibility matrix holds a major release: a pull request labelled `release:major` runs every entry of the [compatibility list](support.md#verified-combinations) on Linux and Windows (`loomux dev compat matrix --full`), and one that touches the list or a fixture runs the entries it changes. So does a change to the code that checks the combinations: the `[[watch]]` tables of the list name such files (the preset and framework tables, the compat scripts and code, the suites of gdUnit4 and GUT) and the entries a change to them runs, all or those of some stacks or addons; a test file selects nothing. Supporting a new runtime or framework version is one added entry in `internal/compat/compat.toml` and the matching row in the support page; a weekly job (`compat-scout`) opens an issue labelled `compat-scout` for each upstream version the list lacks. See [`loomux dev compat`](cli-reference.md#loomux-dev-compat-matrixrunscout-flags).

## Beta releases

A beta is cut by hand from any branch, tag or commit, without a pull request:
run the `release` workflow with `mode=beta`, `ref` (what to build) and `bump`
(`major`, `minor` or `patch`, default `minor`), for example
`gh workflow run release.yml -f mode=beta -f ref=my-branch -f bump=minor`. It
publishes `vX.Y.Z-beta.N` as a pre-release for the version the next stable
release would take, `N` one above the highest beta of it. It commits nothing
and leaves the changelog alone; the tag is its only trace. Install one with
`loomux upgrade --beta` (the newest release, betas included) or
`loomux upgrade --version <x.y.z-beta.n>`.

**Start a beta only from a branch, tag or commit whose code you have
reviewed.** The workflow builds that ref and runs the binary it built while the
step holds the token of the `loomux-release` app, which both rulesets let
through. Code in a ref nobody reviewed could use that token to push to
`master` or to create `v*` tags.

How a machine follows the beta channel is in
[Updates and channels](install.md#updates-and-channels).

## Opening a pull request

Nobody commits to `master`; every change goes through a pull request. The
hooks in `.githooks` refuse a commit on `master` and a push to it once
`go run ./cmd/loomux init --yes` has armed a fresh clone (by hand:
`git config core.hooksPath .githooks`). With an LLM, the `release-pr`
skill does the steps below; by hand:

1. Group the commits by theme, one commit per change. Fold a later
   correction of something this branch introduced into the commit that
   introduced it; only a fix of a bug that was already on `master` keeps its
   own commit. Rewriting a pushed branch needs
   `git push --force-with-lease`. Every message is a Conventional Commit
   (`feat: …`, `fix: …`, `docs: …`; `!` for a breaking change). The
   `commit-msg` hook checks this locally, and the `pr-label` check refuses
   a pull request with any commit header outside the form.
2. Pick one label from the table in
   [How a pull request becomes a release](#how-a-pull-request-becomes-a-release);
   between two levels take the higher.
   It is never lower than the commits: `!` needs major, `feat` minor, `fix`
   patch.
3. Write the body:
   ```
   Release: <level> — <one-sentence reason>

   ## Summary
   - <what changes for a user>

   ## Changelog
   ### Added
   - <entry>
   ```
   Changelog headings are only `Added`, `Changed`, `Deprecated`, `Removed`,
   `Fixed`, `Security`; with `release:none` the section is left out.
4. Check it: `go run ./cmd/loomux dev release parse-body --labels release:<level> --body <file>`.
5. Push the branch, then `gh pr create --base master --label release:<level> --body-file <file>`.

## Setting up releases (maintainers)

1. Labels:
   ```sh
   gh label create release:major --color B60205 --description "Breaking change"
   gh label create release:minor --color 0E8A16 --description "New feature, compatible"
   gh label create release:patch --color 1D76DB --description "Bug fix or dependency update"
   gh label create release:none --color CCCCCC --description "No release"
   ```
2. GitHub App (Settings → Developer settings → GitHub Apps → New): name
   `loomux-release`, webhook off, repository permissions `Contents: Read and
   write`, `Pull requests: Read-only`, `Metadata: Read-only`,
   `Administration: Read-only` (lists the runners, see step 4), "Only on this
   account". Generate a private key, install the app only on
   `xidus90/loomux`, then:
   ```sh
   gh secret set RELEASE_APP_CLIENT_ID --body <client-id>
   gh secret set RELEASE_APP_PRIVATE_KEY < loomux-release.private-key.pem
   ```
3. Rulesets: `.github/rulesets/master.json` for `master` and
   `.github/rulesets/tags.json` for tags `v*`. Both let the `loomux-release`
   app (its App ID, not the client ID, in `actor_id`) and repository admins
   through: the app commits and tags each release, and an admin can fix a
   broken release by hand. Everyone else reaches `master` only through a
   pull request with the status checks `gate-windows` and `build-linux`
   (workflow `ci`) and `check` (workflow `pr-label`) green and an approval
   by the code owner (`.github/CODEOWNERS`) of its last push. The checks run
   the pull request's own workflows and gate, so it can make them pass; the
   approval is what holds, and a push after it needs a new one. That holds
   only while nobody but the code owner and the repository admins has write
   access (today `.github/CODEOWNERS` names only `@xidus90`): the key from
   step 2 is a plain repository secret, and `ci` and `pr-label` hand it to the
   workflows of every pull request from a branch of this repository. A
   collaborator with write access can read it there before any approval and
   mint a token that passes both rulesets.
   Create both, then check that `master` lists rules:
   ```sh
   gh api -X POST repos/xidus90/loomux/rulesets --input .github/rulesets/master.json
   gh api -X POST repos/xidus90/loomux/rulesets --input .github/rulesets/tags.json
   gh api repos/xidus90/loomux/rules/branches/master
   ```
   A changed file goes to `gh api -X PUT repos/xidus90/loomux/rulesets/<id>`.
4. Runners: every workflow first runs `pick-runner` on a GitHub-hosted
   runner, which sends each job to the self-hosted runner of its OS while one
   is online and to GitHub's otherwise. It lists the runners with the app's
   token, so without `Administration: Read-only` everything runs on GitHub's;
   a fork pull request gets no secrets and runs there as well. Its own
   workflow files can still name the self-hosted runners, so every run from
   outside waits for approval:
   ```sh
   gh api -X PUT repos/xidus90/loomux/actions/permissions/fork-pr-contributor-approval -f approval_policy=all_external_contributors
   ```

If a release was dropped, run the `release` workflow by hand
(`gh workflow run release.yml -f pr=<number>`). It refuses a pull request
not merged into `master`, and a rerun after the `chore(release): v*` commit landed
reuses that commit. If a tag or release already exists for that pull
request, do not dispatch again; fix the existing release by hand.
