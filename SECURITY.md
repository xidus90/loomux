# Security policy

## Supported versions

Only the latest release receives security fixes. A fix ships as a new
patch release; older versions are not patched.

## Reporting a vulnerability

Report it privately through GitHub:
<https://github.com/xidus90/loomux/security/advisories/new>.
Do not open a public issue, pull request or discussion for it.

Include the loomux version (`loomux --version`), the operating system, the
steps or input that reproduce it, and what an attacker gains.

loomux is maintained by one person. Expect an acknowledgement within a week.
Once a fix is released, the advisory is published and credits the reporter
unless they ask not to be named.

## Scope

In scope:

- A way past the guard: a write, command or push the policy should refuse
  but lets through, including writes to `.loomux/config.toml`.
- Reading or writing files outside the paths a command is meant to touch.
- Code execution from data loomux reads: configuration, flows, wiki pages,
  hook payloads, recordings.
- The release pipeline and the published binaries.

Out of scope: vulnerabilities in the agent CLIs that call loomux (Claude Code,
Antigravity) or in third-party tools it starts, unless loomux makes them
exploitable.
