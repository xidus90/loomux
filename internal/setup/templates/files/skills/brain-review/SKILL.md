---
name: brain-review
description: Work on an open review case and write a proven proposal. Applies to "review cases", "what is in the review centre", "work off case X", "write a proposal", "a source has changed". Not for taking in a new source (brain-ingest) and not for pulling along after a merge (brain-land).
---

# Review: turn a case into a proven proposal

A review case arises when a source under a wiki page has moved. The code has
formed the case and put an analysis package beside it; this procedure writes
exactly one file — `proposal.md` beside the case. Nothing is decided:
`loomux approve` stays with the human.

## Three rules above everything else

**A single file is written.** `proposal.md` in the case directory.
`case.toml` and `package.md` are the code's bookkeeping and the evidence
against which is checked; the write barrier refuses both, and that is no
obstacle but the point of it. This procedure never writes into the wiki —
the code does that after approval.

**No second attempt — at the evidence.** If **every** claim fails the
evidence check, the proposal is discarded **and the case is kept as manual**:
the skill gets no second run. The human does — with
`loomux approve <id> --amend <file>` they approve a version corrected by hand,
and the case records that as a stage of its own. This applies to exactly this
one outcome.
If the proposal fails at the diff instead — no `diff` fence, no hunk header,
context does not match —, the case stays open and a second, formally correct
proposal is possible. Both are expensive, only one is final, and the form
decides which of the two happens.

**Proven means quoted.** Every claim names a segment number from the package
and carries the quote verbatim in the fence. Verbatim means: copied from the
package, not typed from memory. An empty quote never counts.

**No subagent.** This procedure runs in the main session — like
`brain-ingest` and for the same reason: the human alongside is the only
safeguard that gets to see the diff at all, and a subagent does not see the
interjections.

## Steps 1 to 5 — the procedure

1. **`loomux cases`** — the queue. One line per case: address, area, target
   page, state. The **address is the directory name**, not the `id` field; if
   the two differ, the command warns and the directory name counts.
2. **`loomux case <id>`** — the whole case: state, trigger, sources, then
   `package.md` and, if already present, `proposal.md`. If it shows
   **`local_only: der Quelldiff dieses Bereichs darf kein Cloud-Modell erreichen — kein Skill-Pfad in einer Wolkensitzung`**
   (the source diff of this area must not reach a cloud model — no skill path
   in a cloud session), the procedure ends here. For such a case the command
   prints neither package nor proposal, it only names where both lie — the
   line alone came too late as long as the package stood under it.
   **`--package` is forbidden here**: the flag would print exactly what this
   switch holds back, and typing it would not be carelessness but the breach
   itself. For a closed area the local model writes the proposal, or a human
   writes it. The line stands even when a `proposal.md` already lies beside
   it — that one is there does not make the area open. It stands as well when
   `case.toml` does not carry the field but the area's current manifest says
   `local_only`, or when the mode cannot be determined at all; then it is
   followed by the line `Datenschutzmodus unbekannt: …` (privacy mode
   unknown).
   If it shows the line beginning with **`manuell:`** (manual: no skill path
   is offered for this case), the procedure ends here as well. This line says
   less than it seems to say: it only means that no skill path is open for
   this case — whatever the reason. That can be privacy, but it can also be a
   used-up attempt: if the evidence binding rejects a proposal, the case
   becomes manual in **every** privacy mode, and the rejected `proposal.md`
   stays lying beside it. Both lines can stand at once; each one on its own
   stops.
3. **Read the package.** It carries numbered segments of three kinds:
   `D` for diff hunks of the source, `W` for paragraphs of the wiki page, `Q`
   for source entries. If the package notes that the **prior state was not
   verifiable**, then nobody knows the old version — and then the proposal
   must not claim what stood there before either.
4. **Write `proposal.md`**, following the schema below.
5. **Hand over.** Say what was proposed and under which address.
   The human calls `loomux approve <id>`, or `--reject`, or `--defer`.

## The schema of `proposal.md`

Frontmatter, then one section per claim. The proposed diff stands **in the
section of its claim**, never under a heading of its own.

````markdown
---
case: ultra-brain-2026-08-27-a4f2
category: fix
confidence: high
uncertainty: none
---

## B1 — The search chain uses only one backbone

What changed, in one sentence, then the evidence.

evidence: D1

```
- old
+ new
```

Proposed diff:

```diff
@@ -12,1 +12,1 @@
-The search chain takes 120 ms.
+The search chain takes 80 ms.
```
````

**The order is not free.** The checker takes as the quote the **first fence
after the `evidence:` line**. If the `diff` fence stands in between — that
is, after `evidence:` and before the quote —, the diff becomes the quote, the
verbatim comparison fails, and the claim falls.

A fence **before** the `evidence:` line is skipped as a quote — but it does
not disappear: it stays in the **body** of the section, and the body is
exactly what the applier collects the diffs from. A `diff` fence there
therefore satisfies the diff condition, and **its hunks are applied**. It
does not only help, it can also harm: two diff fences in the same section are
two change sets, and both go through. The reliable sequence is therefore:
`evidence:` line, then the quote fence, then exactly one `diff` fence.

Both belong under `## B1`. The checker reads the body of exactly this section
and takes the diff from it.

If a run of backticks is contained in the quote itself, the fence is chosen
longer than the longest run in it — four backticks around a quote with three.

## The diff is a real unified diff

Five conditions, all checked by the applier, none of them lenient — and
**each single one discards the whole proposal**, not only the one claim:

- **The fence carries the info string `diff`.** A fence without it is no diff
  for the applier. The start is checked, so `diffX` counts as well — that is
  leniency nobody should rely on. If a passed claim carries **no** `diff`
  fence, the proposal is discarded. Whoever proposes to change nothing does
  not write the claim in the first place.
- **Every block begins with a hunk header** of the form `@@ -12,1 +12,1 @@`.
  Text before the first header is a form error.
- **The line numbers count from 1 over the whole file**, frontmatter
  included — not from the first paragraph and not from the segment.
- **Context must match verbatim. There is no fuzz.** The hash latch has made
  sure beforehand that the page is byte for byte the one written against; a
  context that does not match is therefore an error in the proposal and not a
  search order.
- **Hunks do not overlap and do not reach past the end of the file.**

A pure insertion is written `@@ -12,0 +13,1 @@` and appends after line 12.
That is why the target page is read and counted before writing — the line
number is never estimated.

A proposal discarded this way is **not** the same as one that failed at the
evidence: the case is **not** marked as manual. It stays in the queue, and a
second, formally correct proposal is possible. It is expensive all the same —
the human got to read the failure.

## The form barrier — what discards the section

Outside fences the checker reads with a **whitelist of permitted line
forms**. What it cannot classify with certainty is an objection, and the
section with such a line is discarded **entirely** — including a claim that
would be proven on its own. The reason is no pedantry: where the checker and
Obsidian read the same bytes differently, the human at the approval gate sees
something other than what the machine checked.

Permitted outside fences are only: the frontmatter at the start of the file,
`#` headings, `evidence:` lines, fence markers, blank lines and running
prose.

Forbidden outside fences are therefore:

| Form | why it is objected to |
|---|---|
| Lists (`-`, `*`, `+`, `1.`) | list markers, not prose |
| Tables (`\|`), block quotes (`>`), callouts | block forms of their own |
| Rules (`---`, `___`, `===`) other than the frontmatter | ambiguous against Setext |
| Four spaces or a tab at the start of a line | indented code |
| `%%…%%` and `<!-- … -->` | the human does not see them |
| `![[transclusion]]` | pulls foreign text into the rendered page |
| Bidi control characters (U+202A–202E, U+2066–2069) | reorder the display without changing a byte |
| A heading that does not follow the claim pattern | such as `## Diff` |

In practice that means: **no prose line begins with** `-`, `*`, `+`, `>`,
`|`, `<`, `#`, `[`, `:`, `=`, `_`, `~`, a backtick or a digit with a period
or parenthesis (`1.` as well as `1)`). If a sentence would otherwise stand
like that, it is rearranged instead of indented.

**The one exception:** a run of **three or more** backticks or tildes at the
start of a line is permitted — that is fence syntax, and the checker
recognises it as such. Two are not.

**A `[[wikilink]]` in the middle of a line passes today** — only the
transclusion and a `[` at the start of a line are objected to. It is still
not written: a proposal is no wiki text, and a link that lands in the diff
when applying links out of a page nobody has checked for it.

## What the checker does with a claim

Three questions per claim, all pure text work without a model:

1. Does the named segment exist in the package?
2. Does the quote occur verbatim in it — exact substring comparison?
3. Does the claim carry any evidence at all?

If **one** claim fails, it is removed and noted in the case; the rest stays.
If **all** fail, the proposal is discarded. That is why one terse, safely
proven claim is worth more than three of which two have lost their context.

## The diff is not checked

Three limits that belong together and that this procedure has to carry
itself, because no code carries them:

- **The form barrier only checks outside fences.** The content of the `diff`
  fence is written literally into the wiki page. Everything the table of
  forbidden forms above brands as "the human does not see them" — bidi
  characters, `%%…%%`, HTML comments, `![[…]]` — would get into the wiki
  unhindered through the diff. It therefore does not stand there either.
  That `loomux case <id>` prints the proposal raw into the terminal softens
  this only in part: an HTML comment is visible there, a bidi control
  character **takes effect in the terminal too** and reorders the line the
  human checks.
- **The evidence is weaker than "proven" sounds.** What is checked is a
  substring occurrence in the one named segment, not whether the quote
  supports the claim. A true half sentence from the wrong context passes this
  check.
- **The hunk numbers are bound to the quoted segment by nothing.** A proven
  sentence about paragraph 3 can carry a hunk that rewrites paragraph 17.

That is the most important limit of this procedure: the evidence binding
checks the quote, **never the diff beside it**. A cleanly proven sentence can
carry a change nobody vouches for. Only the human who reads both catches
that — and they read what `loomux case <id>` prints. The diff therefore stays
as small as possible and changes nothing that no claim proves.
