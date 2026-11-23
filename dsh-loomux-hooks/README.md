# dsh-loomux-hooks

Native DeepSeek Harness hooks backed by loomux: the write barrier, the check
chain and the Stop guidance of a project run in every harness session, as
real hook processes — no bridge, no MCP round-trip.

The plugin registers the six harness seams, resolves the hook entries per
session directory along the same chain loomux documents (`configPath`,
`.loomux/dsh/hooks.json`, `.claude/settings.json`, `hooks/hooks.json`), runs
each matching command exactly the way Claude Code runs hooks, and turns the
exit code and JSON answer into the harness-native action: refuse, block,
steer, inject.

- An internal failure never locks: exit 1, a timeout and unparseable output
  let everything through; a timeout kills the whole process tree.
- Loop-protected: `stop_hook_active` marks repeat evaluations and at most
  eight Stop steerings run per agent.
- Every hook run appends the known `hook/invoked` and `hook/result` records
  to the session log; unreachable log, the plugin logger writes instead.
- No hooks file for a session's directory: the seams stay silent, and that
  session runs unhooked.

## Configuration

```yaml
- id: loomux-hooks
  name: dsh-loomux-hooks
  config: {} # or:
  # config:
  #   configPath: C:/work/proj/.loomux/dsh/hooks.json # skip the chain
  #   defaultTimeoutMs: 60000 # per hook entry without a timeout
```

`binary` is accepted and ignored: each hook command carries its own binary
path, written by `loomux init --hosts=dsh`.

## Requirements

- Node.js >= 22.19
- `loomux` built from this repository; `loomux init --hosts=dsh` writes the
  hook file `.loomux/dsh/hooks.json` the plugin resolves by default

The full guide, including the seam table and the failure behaviour, is in
[`docs/en/dsh-hooks.md`](../docs/en/dsh-hooks.md).

## Release

`sh ci/dsh-loomux-hooks.sh` runs the tests; the workflow runs it on every
pull request that touches the package. `.github/workflows/dsh-loomux-hooks.yml`
publishes the package: a tag `dsh-loomux-hooks-v<version>` runs the tests
and then `npm publish --provenance`, which needs the repository secret
`NPM_TOKEN` and a tag that matches the version in `package.json`.

## License

[PolyForm Noncommercial 1.0.0](LICENSE.md), the license of the loomux
repository; `LICENSE.md` here is a copy of the one at its root.
