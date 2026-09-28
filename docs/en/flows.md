# Flows

A flow is a graph of nodes written as data: loomux runs it, pauses it at a
gate until a human answers, resumes it and replays it from its journal. The
commands are `loomux flow run|resume|replay|show|list`; every flag and exit
code is in the [CLI reference](cli-reference.md#12-flows-loomux-flow).

**What runs today.** Gate and exit nodes run. Agent nodes load, show their
role and model, and run in the catalog's test against a fake model, but
loomux has no model adapter yet: a flow with an agent node refuses to start
with `no adapter for provider claude yet` (or the provider its roles resolve
to), before a run exists. The adapters, flows over MCP and further node kinds
are on the [roadmap](../../README.md#roadmap).

---

## 1. What a Flow Is

A flow is a folder, and the folder's name is the flow's name:

```
<name>/
  flow.toml            # schema_version = 1
  instructions/*.md    # what the agent nodes are told
  questions/*.md       # what the gates ask
  README.md            # required in the catalog only
  _test/               # catalog only: script.toml and journal.jsonl
```

- **Flow names** follow `[a-z][a-z0-9-]*`: lower case, hyphens allowed, a
  letter first (`dev-cycle`, `strict-security-review`). Windows does not tell
  `Review` from `review`, and `go:embed` leaves out a name that starts with
  `_` or `.`. Node, field, parameter and role names follow
  `[A-Za-z_][A-Za-z0-9_]*`; a node, field or parameter may not be named
  `true`, `false` or `END`.
- **`flow.toml`** holds `schema_version` (this build reads `1`), `[flow]` with
  `start` (the first node, required) and `role` (the role of every agent node
  that names none), `[params]`, `[state]`, `[[node]]` and `[[edge]]`. An
  unknown key at the top, in `[flow]`, in a node or in an edge is a load error
  that names the known keys. `[flow] name` is one too (the folder gives the
  name), and so is `model` in `[flow]` or a node (a flow names a role, see
  section 3).
- **Parameters and state fields** are declared as
  `name = { type = "…", default = … }` with the types `string`, `int`, `bool`
  and `list[string]`. Every one needs a default; a name may not be both a
  parameter and a field. Parameters are what a run is started with
  (`--option name=value`); fields are what nodes write. A parameter may not
  take a name the run's marker keeps for itself — `baseline`,
  `baseline_commit`, `loomux_version`, `origin`, `overlays` — since its option
  could never be written there (`parameter "origin" is a name the run's marker
  keeps for itself; name it otherwise`).
- **Texts.** An instruction is a file under `instructions/`, a question a
  file under `questions/`, named in `flow.toml` relative to the folder with
  `/` between folders. Any other place, an absolute path or one that leaves
  the folder is a load error: a fixed place per kind is what lets a project
  overlay a text file by file (section 4). A carriage return before a line
  break is dropped on reading, so a file saved with CRLF renders and hashes
  like its LF twin. `{{name}}` renders a field or a parameter; a list renders
  as one `- ` line per entry.

### Node kinds

Every node has `name` and `kind`, and may have `max_visits`; an agent node may
have `role`.

| Kind | Keys | What it does |
|---|---|---|
| `agent` | `instruction` (required), `reply` (required: at least one field, each `string`, `int` or `bool`), `tools` (`read_only`, the default; `edit`; `shell`; `mcp`), `effort` (text), `role` | Renders the instruction, asks the model its role resolves to, and checks the answer against `reply`; the reply's fields are written into the state. |
| `gate` | `question` (required), `choices` (required: at least two texts without blanks or `:`, none twice), `answer` (required: a state field) | Pauses the run and asks. An answer is a choice, or a choice followed by a blank or `:` and a reason (`no: too thin`); upper and lower case count. The choice goes into the field `answer` names, the reason into `<answer>_text`; `[state]` declares both as `string`. |
| `exit` | `code` (required: 2 or 4 to 255), `message` (required, inline text with placeholders) | Ends the run with its own exit code and message. 0, 1 and 3 are refused: the runtime spends them on done, failed and paused. |

`kind = "command"` is refused at load with `command nodes are planned but not
built`; any other unknown kind names the known ones.

The tool profiles: `read_only` is Glob, Grep and Read; `edit` is `read_only`
plus Edit and Write; `shell` is `read_only` plus Bash, with no Edit or Write;
`mcp` is `read_only` plus one `mcp__<server>` for each server in
[`[agent] mcp_servers`](configuration.md#agent-models-for-flow-roles). Any
other name is a load error (`unknown tool profile "bogus"; known profiles:
edit, mcp, read_only, shell`).

### Edges and conditions

An edge has `from`, `to` (a node, or `END`), and at most one of `when` and
`on_error = true`.

- **The first edge that holds is taken.** After a node, loomux reads its
  edges in file order and takes the first whose `when` holds; an edge without
  `when` always holds. Where none holds, the run ends as an error (`no edge out
  of "x" applies to the current state`).
- **An error edge** is taken when the node fails, carries no `when`, and is no
  way out on the normal path: a node needs at least one ordinary edge.
- **A condition** is `<field> <op> <value>` with `==`, `!=`, `<`, `<=`, `>`,
  `>=` (ordering only for `int`), a `list[string]` field compared only with
  `[]`; several terms are joined only with `|` or only with `&`, without
  brackets. Every name and type is checked at load.
- **Visits.** `max_visits` is 1 when absent, an integer of at least 1, or
  `"<int parameter> + <n>"`. A node on a cycle must allow more than one visit.
  A visit beyond the ceiling ends the run as an error (`node "draft" exceeded
  max_visits=6`).

### The load check

A flow loads in stages: its declarations; every node kind and its keys; its
texts; every name a text, a condition or a ceiling uses; every field a node
writes against `[state]`; then the graph (the start exists, every edge ends at
a node or `END`, every node is reachable and has an ordinary way out, no node
on a cycle allows one visit). The first stage with findings stops the load and
reports all of them, one line each behind the file name.

---

## 2. Running a Flow

```sh
loomux flow run [<flow>] [--option name=value]... [--root dir]
loomux flow resume <run> [--answer text] [--root dir]
loomux flow replay <run> [--root dir]
loomux flow show <run|flow> [--root dir]
loomux flow list [--root dir]
```

- **`run`** starts a new run of the flow, or of `[flow] default` when no
  name is given. Without a default it refuses and names the flows it knows.
- **`resume`** carries a paused run on. With `--answer` it hands the gate its
  answer; without, the gate asks again and nothing is written. The answer is
  a human's: the guard refuses it to an agent (section 6).
- **`replay`** re-derives a finished run from its journal and executes
  nothing.
- **`show`** prints a run's journal (a run number is digits) or a flow's
  nodes, roles and edges (a flow name starts with a letter).
- **`list`** names every flow the project can run, with its origin, and a flow
  that will not load with the reason instead of leaving it out.

The project is `--root`, else the nearest `.loomux/config.toml` above the
working directory, else the working directory itself: a project without a
configuration still has flows.

**Exit codes:** `0` done, `1` error or refusal, `2` usage error, `3` paused at
a gate, and an exit node's own code. A run that paused tells the gate's
question on stdout:

```
$ loomux flow run ship
run 0001 (ship, project): paused
Ship it?
$ echo $?
3
```

**Where runs live.** Under `.loomux/state/runs/` of the project, which is
git-ignored with the whole state folder, so every checkout and every worktree
has runs of its own. A run is a four-digit number, one more than the highest
there, and two files:

- `<id>.jsonl`, the journal: one line per step with the node, its kind, the
  outcome (`ok`, `paused`, `error`), the delta it wrote, tokens and seconds,
  the model as `<provider>:<model>` or `<provider>:cli-default`, the role, and
  two hashes — one over what the node was told (`definition_hash`), one over
  the state it saw (`input_hash`).
- `<id>.flow`, the marker: the flow's name on the first line, then the
  options, where the flow came from (`origin`, and `overlays` for an overlay),
  the commit and the changed files the run started from (`baseline_commit`,
  `baseline`, when git answers), and the `loomux_version` that started it.

A resume and a replay read the flow the marker names. When its `flow.toml`
now comes from the other source — the project folder instead of the catalog,
or the other way round, say because `[flow] overrides` changed — they refuse
and name both origins, since the open gate may belong to another graph:

```
run 0001 started on project (hides bundled) and example now resolves to bundled, another flow.toml; start a new run with loomux flow run example
```

A different set of overlay files is the same graph with other texts: they warn
and go on, and a node whose entry in `flow.toml` (its own `role` among it),
instruction, model, effort or tools changed is named as having run on a
definition that has changed since. The role counts through the model it
resolves to: a changed binding changes the definition, and so does a node's
own `role`, but a changed `[flow] role` only when it resolves to another model.

A run reads no stdin and keeps its whole state in journal and marker, so a
second caller — a shell of an agent today, an MCP tool later — drives it the
same way. A run that was cut off between two nodes (by hand, a crash, a tool's
time limit) has no open gate and no end, and `resume` refuses it.

---

## 3. Roles and Models

A flow names roles, never models; the project binds them. For each agent node
the first step that is set wins:

1. `role` on the node, else `[flow] role`.
2. The binding of that role in `[agent.roles]` of `.loomux/config.toml`: a
   model name.
3. Without a role or without a binding: `[agent] default`, also a model name.
4. Nothing set: provider `claude`, on the CLI's own default model.

A model name points at `[agent.models.<name>]` with `provider` and optionally
`model`; without `model` the provider's CLI picks its default. The keys are in
the [configuration reference](configuration.md#agent-models-for-flow-roles).
A flow therefore loads in every project, and a role a project binds that no
flow names is no error: a project binds roles for several flows.

A run holds one model, so the agent nodes of one flow must resolve to one
provider; a flow whose roles resolve to two is refused (`flow "x" asks agy and
claude; a run holds one model, so a flow asks one provider`).

`loomux flow show <flow>` prints each agent node with its role, the model it
resolves to and how:

```
$ loomux flow show example
example (bundled)
nodes:
  draft    agent  writer  claude:cli-default  role writer (node), the CLI's own default
  approve  gate
  stop     exit
edges:
  draft -> approve [verdict == "done"]
  draft -> draft
  approve -> END [answer == "yes"]
  approve -> stop
  stop -> END
```

With `[agent.models.w] provider = "agy"`, `model = "m1"` and
`[agent.roles] critic = "w"`, a node on the flow's role `critic` reads:

```
  judge  agent  critic  agy:m1  role critic (flow), bound to w
```

One covered by `[agent] default` ends in `[agent] default <name>`. The resolved model goes into the definition hash, so a
changed binding shows up on a replay as a changed definition.

---

## 4. The Catalog and Overrides

Flows come from two places:

- **bundled** — the catalog under `flows/catalog/` of the loomux repository,
  compiled into the binary. Today it holds one flow, `example` (an agent
  drafts, a human approves, a rejection ends with code 4), the template to
  copy.
- **the project** — `.loomux/flows/<name>/`.

A project folder with a `flow.toml` is a **flow of its own**. A folder with
only `instructions/*.md` and `questions/*.md` is an **overlay** over the
bundled flow of the same name: each file replaces the bundled file of the same
path. An overlay file the bundled flow does not have, an overlay without a
bundled flow of its name, and any other file beside an overlay are load
errors: the first two say `… overlays nothing: …`, the third `… is neither
flow.toml nor a file under instructions/ or questions/`. A flow of its own may carry the `README.md`
and `_test/` a catalog flow has, so a copied template loads unchanged. Names
starting with `.` are passed over.

**Only a human lets a project flow take a bundled flow's place.** A project
folder named like a bundled flow counts only when `[flow] overrides` in
`.loomux/config.toml` names it. Otherwise the bundled flow runs, and `list`,
`show`, `run` and the session start say so:

```
warning: .loomux/flows/example is ignored: [flow] overrides does not name it
```

The origin every command prints, and the session start with it:

| Origin | Means |
|---|---|
| `project` | a flow of the project's own, under a name the catalog does not have |
| `bundled` | the catalog's flow, as shipped |
| `bundled+overlay: <files>` | the catalog's flow with the listed files from the project folder |
| `project (hides bundled)` | a flow of the project's own under a catalog name that `[flow] overrides` names |

If a release brings a bundled flow under a name a project already uses for a
flow of its own, the bundled one runs from then on, with the warning above,
until `[flow] overrides` names it.

`[flow] default` picks what `loomux flow run` starts without a name. A default
that names no flow fails `list` and a `run` without a name (`[flow] default
names "x", which is no flow here`), and `list` warns about a name in
`overrides` the catalog does not ship. The config check reads no flows, so
both surface only there.

---

## 5. Contributing a Flow

A contribution to the catalog is data only:

1. A folder `flows/catalog/<name>/` with `flow.toml`, `instructions/`,
   `questions/` and a `README.md` in English: what the flow does, the roles it
   names, the parameters it takes.
2. `_test/script.toml`: the run's options, then one step per visit that needs
   an answer from outside, in the order the run asks. An agent step names its
   node and carries the reply fields (and `tokens`), or an `error` the fake
   model raises instead; a gate step names its node and carries the `answer`.
   Unknown keys fail the test. The `example` flow's script:

   ```toml
   # The run drafts once, asks, and is rejected.
   [[step]]
   node = "draft"
   reply = { verdict = "done", count = 2 }
   tokens = 120

   [[step]]
   node = "approve"
   answer = "no: too thin"
   ```

   Options go into a table `[options]` of text values, as
   `--option` would give them.
3. `go test ./flows -update` writes `_test/journal.jsonl`, the golden
   journal; read it. `go test ./flows` then loads every catalog flow as the
   binary ships it, runs it against the fake model along its script with a
   fixed clock, and compares the journal byte for byte. A missing
   `README.md`, `_test/script.toml` or `_test/journal.jsonl` fails the test.
4. A pull request with the label `release:minor`: a new bundled flow is a new
   feature.

`_test/` never reaches the binary — `go:embed` leaves out names that start
with `_`. A flow that needs a node kind the runtime lacks is a separate pull
request with Go code.

---

## 6. Gates Are a Human's

A gate asks a human, and the guard (`loomux hook pre-tool-use`) keeps the
answer and everything that could replace it away from an agent:

- **The answer.** `loomux flow resume … --answer` is refused to an agent in
  every spelling the rule for `loomux config` reads — a path to the binary,
  quotes, chained commands, `--answer text` and `--answer=text` — and so is a
  `Start-Process` of loomux, whose arguments the guard cannot see:

  ```
  a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer "…"` themselves
  ```

  `run`, `show`, `list`, `replay` and `resume` without `--answer` stay open to
  an agent. `resume` and `replay` take only a run number of digits, so a
  journal and marker an agent forged elsewhere in the project (`resume
  ../../mine/x`) cannot stand in for a run.
- **The run files.** `.loomux/state/runs/**` under any directory is held by
  the path rules, for a writing tool and a shell line alike: an `answered`
  entry an agent wrote into a journal would hand the next resume an answered
  gate. A removal of a folder above it, and a glob that matches it on disk,
  count as a write.
- **The bundled flows.** A write under `.loomux/flows/<name>/`, under any
  directory, is refused by the path rules, for a writing tool and a shell line
  alike, when `<name>` is a flow of this binary's catalog or is named in
  `[flow] overrides`, in any case of the name. A glob in the name's place
  counts where it matches such a folder on disk (`cp x .loomux/flows/ex*/`),
  and a removal of that folder or of `.loomux/flows` above it counts too; an
  agent writing a flow of its own spells its name. A flow under a name of its
  own an agent may write and run; it
  cannot make it the default, because `[flow] default` stands in
  `.loomux/config.toml`.

The exact rules are in [Hooks](hooks.md#7-the-decision-path-of-pre-tool-use).

**How the human answers.** The session start names every waiting run with
the command to answer it, built from the path of the binary the hook runs
from, with forward slashes (loomux is not on every `PATH`), in double quotes
when it holds a blank or another character a shell reads:

```
run 0001 (ship, project) is waiting at confirm: Ship it?
  a human answers it with: C:/Users/me/project/bin/loomux.exe flow resume 0001 --answer "your answer"
```

With an overlay, the origin names the files it replaced, as in `(example,
bundled+overlay: questions/approve.md)`.

The human runs that line in a terminal of their own, or in Claude Code with
the `!` prefix, which runs it in their shell as their command rather than as
the agent's tool call. The refusal names no path of its own: which command
works for the human depends on their terminal and binary, and the notice has
the one this project's hooks use.

**The limit, named.** The guard reads tool calls, not programs: a program an
agent writes and then starts can do what the guard refuses. It knows loomux by
its file name, `loomux` or `loomux.exe` (or `go run` of `cmd/loomux`): a
copied or renamed binary passes every rule on loomux's own commands, the
refused answer included. The gates hold an agent that keeps to its tools; the
journal records every answer with its time and visit.
