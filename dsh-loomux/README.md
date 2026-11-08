# dsh-loomux

The `grep` tool of a DeepSeek Harness profile, served by `loomux graph grep`.

The plugin is a facade: it maps the tool arguments to one `loomux graph grep
--json` call, always on the cloud channel, and shapes the answer back into
matches with line numbers and, when there is one, the enclosing symbol.

- One search corpus for every text file git does not ignore; ignored, binary
  and oversized files are counters in the answer, not an error.
- The trigram index answers patterns with a literal; a scan answers the rest.
- A failure is an error with a hint, never an empty list: missing binary,
  missing graph, invalid or unsupported pattern, prefix outside the corpus,
  cancellation and an answer beyond the output cap all fail.
- No shell, no fallback to another search tool, no second code path.

## Configuration

```yaml
- id: loomux
  name: dsh-loomux
  config:
    binary: loomux # resolved through PATH, or an absolute path
    maxResults: 300 # hits per call when the model gives no max_results; 20–5000
    trustIndex: false # true adds --trust-index to every search
```

`include` takes one glob without brace alternation (`*.go`, `src/**/*.ts`):
loomux matches globs without expanding `{a,b}`, so the plugin rejects one
instead of answering with an empty list.

`trustIndex` (default `false`) makes every search answer from the index without
checking the tree against it. Turn it on where the tree does not change after
the build: CI, a fresh checkout. It is a plugin option only; the agent cannot
set it per call. A tree that did change after the build then answers from the
old index, and the answer names the build time of the index it trusted.

## Requirements

- Node.js >= 22.19
- `loomux` with the `graph grep` subcommand (built from this repository)
- A built graph in the workspace: `loomux graph build`

## Release

`sh ci/dsh-loomux.sh` runs the tests; the workflow runs it on every pull
request that touches the package. `.github/workflows/dsh-loomux.yml` publishes
the package: a tag
`dsh-loomux-v<version>` runs the tests and then `npm publish --provenance`,
which needs the repository secret `NPM_TOKEN` and a tag that matches the
version in `package.json`. A version bump without that tag publishes nothing,
and a tag without the matching version stops before the registry is touched.

## License

[PolyForm Noncommercial 1.0.0](LICENSE.md), the license of the loomux
repository; `LICENSE.md` here is a copy of the one at its root.
