---
name: brain-research
description: Get to the bottom of a topic across the stocks. Applies to "what do we know about X", "research X", "is there anything on this already", "get to the bottom of topic X" — questions that look for existing knowledge instead of taking in new knowledge. Not for working in a source; that is brain-ingest.
---

# Research: a topic across the stocks

The search runs **in a subagent**, the assessment in the main session. That is
no formality: hit lists are the cost driver of this system, and what the
subagent reads does not weigh on the main conversation.

## The subagent

Dispatch with **`model: opus` and `effort: low`, both set explicitly.**
Otherwise inheritance silently falls back to the session value — a mistake
that goes unnoticed because the result still arrives, only more expensively.

Its task is the search ladder (architecture §8), in this order:

1. **`loomux brain catalog`** — root catalog, if needed exactly one area catalog (`loomux brain catalog --scope <area>`).
2. **`loomux brain search "<query>"`** — does the knowledge already exist? Optionally with `--scope <area>`.
3. **`loomux brain read <file> --scope <area>`** — exactly ONE file, and in it only the relevant section (`--section <section>`).

## The limits that must be in the task

- **At most two search runs per question.** Every further step reads the
  conversation so far again; the costs grow from round to round. A stop
  criterion is cheaper than any optimisation of the search.
- **A high relevance score is no proof.** What counts is the passage read,
  not its rank.
- **Widen the scope only deliberately — and name the widening.**
- **Quote contradictions, do not resolve them.**

## The fixed reporting task

The subagent delivers:

- **Evidence with source and location.** Scope, path, section — enough that
  the main session can look up every piece of evidence without a second
  search.
- **A section of its own, "not found".** Mandatory, even if it stays empty.
  Without it the gap disappears silently, and research that keeps quiet about
  its own limits reads more complete than it is.

## The main session

It summarises and **asks** whether the result should be filed in the wiki as
a `Synthesis` with `status: draft`. **Without consent nothing is written.**
Research is an answer; whether it becomes stock is a second decision.

## When filing

The area is named, not guessed — with several bundles, mixing them up is the
most likely mistake.

1. Page as `type: Synthesis`, `status: draft`.
2. `sources[]` complete: `doc_id`, `content_hash` and `revision` per piece of
   evidence.
3. A line in the bundle's `index.md` — otherwise the check reports
   `house/orphan`, and from the catalog page `okf/catalog-malformed`.
4. A line in `log.md`.
5. `loomux brain check bundle --scope <area>` — if it reports an error, the
   page is not finished. Warnings let the run pass.

**The reading direction applies here too:** projects read shared areas,
shared areas (`knowledge`, `engineering/*`) do not read projects. A synthesis
in a shared area that cites a project page fails as `house/wrong-direction`.
The way there is promotion by human hand, not the quote.

**This rule runs along in the bundle width.** It and `house/unlisted-area` do
not need all bundles, but the registry; step 5 therefore checks them for the
area it names. An additional `loomux brain check all` brings nothing new for
this one area — it only says the same about all the others.

**If either of the two runs answers with 2, it did not take place** — that is
no finding about the page and must not be read as "clean". The reason is on
stderr:

```
error: area '<area>' has no wiki at <path>; run `loomux wiki init --scope <area>` first
error: no area named '<area>' in the registry
```

`loomux brain check all` already hits this when **a single** registered area
is in that state: the wide run then aborts entirely, for the healthy ones as
well. One caveat about the advice of the first message: if the path points
into another checkout, that checkout is mostly just on a branch without this
directory — then the checkout has to be switched, not a bundle created. Until
then `loomux brain check bundle --scope <area>` from step 5 carries on in
full: the two federation rules run along there, and only the areas the wide
run no longer reaches stay unchecked.
