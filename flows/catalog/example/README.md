# example

The template to copy for a flow of your own: an agent drafts, a human
approves, a rejection ends the run with code 4.

- **Roles:** `writer` (the node `draft`). Bind it in `.loomux/config.toml`
  with `[agent.roles] writer = "<model>"`; unbound, it runs on `[agent] default`
  or the claude CLI's own default.
- **Parameters:** `max_rounds` (int, default 5).
- **Gate:** `approve`, choices `yes` and `no`.

Until loomux has a model adapter, `loomux flow run example` refuses to start;
the flow runs in its test, `flows/catalog/example/_test/`.
