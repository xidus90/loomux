---
name: brain-ingest
description: Condense a source into the wiki. Applies when a note, a document, a protocol or a measurement is at hand and is to be worked into the wiki of an area — "condense X", "work X into the wiki", "take X in", "this belongs in the wiki". Not for research over existing knowledge; that is brain-research.
---

# Ingest: condense a source into the wiki

The procedure is the same in every session (architecture §9.2). It is short
because it has to stay cheap: step 3 makes sure the whole wiki is never read.

## Two rules above everything else

**Raw sources stay untouched.** `10 Rohquellen/` (the raw sources) in the
vault and the raw sources of the code repos are read, never written. What is
condensed becomes a new page in the wiki bundle — the source stays as it is.

**No subagent.** This procedure runs in the main session. The human alongside
is the only safeguard here, and a subagent does not see the interjections.

## Step 0 — name the area

There are several bundles, and mixing them up is the most likely mistake of
this procedure. **If the area is not named, ask** before you read anything.
`loomux brain catalog` lists the available areas.

Then read the area's `_schema.md`. It is editable per bundle, and what it says
takes precedence over everything written here.

## Steps 1 to 8 — the procedure

1. **Read `_schema.md`** (step 0 has already done so).
2. **Read the source**; determine `doc_id`, `content_hash` and `revision` —
   through `loomux brain read <file> --scope <area>` or the identity register
   of the source area.
3. **Find the affected pages through the wiki catalog.** Read the bundle's
   `index.md`, do not search the bundle. This step is the reason ingest stays
   affordable over a large stock.
4. **Open exactly these pages — never the whole wiki.**
5. **Write the source page**; update topic and entity pages, create missing
   ones, set links.
6. **Check each page** — see the stop rule below.
7. **Record `sources[]`** with `doc_id`, `content_hash` and `revision` per
   piece of evidence. If one of them is missing, the check reports
   `house/source-incomplete`; if `resource` is missing,
   `okf/source-resource-missing`; if `sources[]` is empty, `house/no-sources`.
8. **Entry in `log.md`**: date, page, what changed, from which source.

## Step 6 — the stop rule

Exactly three possibilities for every statement:

- **It fits** → work it in.
- **It contradicts** → set a conflict box, increase `open_conflicts`,
  **overwrite nothing**.
- **It is unclear** → mark it as well. **When unclear, mark, do not
  decide.**

The form of the box is fixed and machine-findable (§9.3):

```markdown
> [!conflict] <short title>
> [Source A](/path/a.md) says X.
> [Source B](/path/b.md) says Y.
> Both states stay in place. Decision open.
```

The four admissible verdicts — old state outdated, both hold in different
contexts, new information is wrong, deliberately left open — are made at
**resolution**, not here. And what is resolved is the source, never the box:
whoever only deletes the marker finds the same contradiction again in the
next pass.

## Step 9 — the lint

```bash
loomux brain check bundle --scope <area>
```

**If it reports an error, the ingest is not finished.** Warnings let the run
pass — the exit code stays 0 —; they are a hint for attention, not a defect.

**Exit code 2 is not a finding but "the run did not take place at all".**
Nothing was read, so it says nothing about the ingest either. The message is
on stderr and names the reason — two cases:

```
error: area '<area>' has no wiki at <path>; run `loomux wiki init --scope <area>` first
error: no area named '<area>' in the registry
```

In the first, the area is registered, but no directory lies under the
recorded path. One caveat about the advice the message gives: if the path
points into another checkout, that checkout is often just on a branch without
this directory — then there is nothing to create, the checkout has to be
switched instead.

In the second, the area name is wrong; `loomux brain catalog` names the
existing ones.

**Do not confuse it with 0 and do not skip it:** 0 means checked and clean,
2 means unchecked.

Every rule name carries its axis up front: `okf/` is format conformance with
the Open Knowledge Format, `house/` are our stricter house rules. An ingest
therefore regularly sees `okf/` warnings the old lint never knew.

The most frequent findings after an ingest and what they mean:

| Finding | Cause |
|---|---|
| `house/conflict-count` | box set, `open_conflicts` not increased |
| `house/orphan` | new page created, but no line in `index.md` |
| `okf/catalog-malformed` | the same gap seen from the catalog page |
| `house/source-incomplete` | `doc_id`, `content_hash` or `revision` missing (an `http(s)://` source needs none) |
| `house/dead-link` | link to a page that does not exist (yet) |
| `house/untouched` | not touched for a long time — warning, not a defect |

**The two federation rules run here too.**
`house/wrong-direction` (a shared area cites a project page) and
`house/unlisted-area` (the signpost does not name a registered area) do not
need all bundles, but the registry — the shared name pool and the list of
areas the signpost has to name. The bundle width passes it through, and a run
over one area says the same about it as `loomux brain check all`.
`house/unlisted-area` only reports when the checked area is the signpost
itself.

## Step 10 — the log entry

Only when the lint is clean, the line in `log.md`. It is the last action, not
the second to last: a log entry about a change that still carries errors
claims something false.
