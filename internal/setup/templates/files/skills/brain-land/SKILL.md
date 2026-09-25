---
name: brain-land
description: After the merge to main, pull the plan onto the actual state. Applies to "the merge is through", "is this implemented now", "pull the pages along", "set realization", "enter implemented_in" — cases with the trigger `merge`. Not for cases from a source change; that is brain-review.
---

# Land: after the merge, pull the plan onto the actual state

A merge to `main` has opened a case for every page that still promises
something. It asks exactly one question: **Is what this page planned built?**
This procedure answers it per page and writes a proposal — pull the wording
along, `realization: implemented`, set `implemented_in`.

Here, too, only `proposal.md` beside the case is written. After approval the
code writes into the wiki; `loomux approve` stays with the human.

## How to recognise the case

`loomux cases` shows it with state **`due`**, `loomux case <id>` names
**`merge`** as its trigger. The difference to the case from a source change
is not only the occasion but the evidence:

| | Source case | Merge case |
|---|---|---|
| State | `source_changed` | `due` |
| Segments | `D` diff hunks, `W` paragraphs, `Q` sources | `D` changed paths and commit subjects, `W` paragraphs |
| Question | is the page still right? | is what it promised delivered? |

**No source text in the package, and that is on purpose.** The evidence of
this trigger is paths and subject lines, nothing more is fetched. What cannot
be shown from them is not claimed.

**No subagent.** As with `brain-review` and `brain-ingest`: the procedure runs
in the main session, because the human alongside is the only safeguard that
gets to see the proposed diff.

## Steps 1 to 5 — the procedure

1. **`loomux cases`**, then **`loomux case <id>`** for the case with trigger
   `merge`. If it shows
   **`local_only: der Quelldiff dieses Bereichs darf kein Cloud-Modell erreichen — kein Skill-Pfad in einer Wolkensitzung`**
   (the source diff of this area must not reach a cloud model — no skill path
   in a cloud session), the procedure ends here. For such a case the command
   prints neither package nor proposal, it only names where both lie.
   **`--package` is forbidden here**: the flag would print exactly what this
   switch holds back, and typing it would not be carelessness but the breach
   itself. For the same reason `package.md` and `proposal.md` under the named
   location are not opened. For a closed area the local model writes the
   proposal, or a human writes it. The line also stands when `case.toml` does
   not carry the field but the area's current manifest says `local_only`, or
   when the mode cannot be determined at all; then it is followed by the line
   `Datenschutzmodus unbekannt: …` (privacy mode unknown).
   `manuell` alone cannot be relied on here: a case of a closed area carries
   `manual = false` as soon as a local proposal exists.
   If it shows **`manuell`** (manual), the procedure ends here as well. Both
   lines can stand at once; each one on its own stops.
2. **Find the page's promise — in the prose, not in the frontmatter.**
   `W` segments are the page's paragraphs separated by blank lines, and the
   package builder **cuts the frontmatter off beforehand**, explicitly:
   otherwise the evidence binding could be satisfied by citing a `doc_id`.
   `realization: planned` is therefore **not a `W` segment and not citable**.
   Citable is the paragraph that makes the promise in words. (The one
   exception is a defect, not a path: if the page lacks the closing `---`,
   the package builder passes the text through unchanged, and the frontmatter
   then does stand in a segment. Whoever exploits that proves a promise
   through a broken header — the page gets repaired, not cited.)
3. **Check against the paths and subjects.** Does a `D` segment name a path
   or a subject line that delivers exactly this promise? If yes, that is the
   evidence. If no, the case is **not** landable — see below.
4. **Write `proposal.md`**, in the same form as with `brain-review`: one
   heading per claim, `evidence: <segment number>`, the quote verbatim in the
   fence and the unified diff with an `@@` header in the same section. The
   form rules there apply unchanged; they are not a second version but the
   same barrier.
5. **Hand over.** What was proposed, under which address. The decision is
   made with `loomux approve <id>`, `--reject` or `--defer`.

## What the diff changes

Three things, and per page at most these three:

- **`realization: implemented`** in the frontmatter.
- **`implemented_in`** with the commit that built it. Without this field the
  check reports `house/implemented-without-commit`, and the promise would be
  unchecked again — exactly the state the field is meant to abolish.
- **The wording**, where the page speaks in the future tense about something
  that stands now. "Shall" becomes "is", and nothing more.

Everything else does not belong here. New insights from the work are an
ingest, not a landing.

## The honest gap: the commit identifier

`implemented_in` names a commit, but the package carries paths and subjects —
**the identifier itself stands in no segment.** It is therefore taken over
from the merge and not gained from a quote. The proven claim is "this promise
is delivered, see this path, see this subject"; the identifier is the address
to it, which the human reads along at the approval gate. If it is guessed, it
is wrong, and the lint never notices — it checks **that** the field is there,
not which commit it names.

## The four outcomes

Exactly four possibilities for every page:

- **Delivered and provable** → write the proposal.
- **Delivered, but the promise stands only in the frontmatter.** A page whose
  whole promise is `realization: planned` and which states it in no paragraph
  is **not provably landable**. It is set aside with `--defer` and pulled
  along by hand or through an ingest; a quote of the frontmatter fails, and
  since it would usually be the only claim, the whole proposal is discarded
  and the case is kept as manual — that is the outcome that knows no second
  attempt.
- **Not delivered** → propose nothing, set the case aside with `--defer`.
  The page stays `planned`, and then that is right, too.
- **Partial or unclear** → **mark, do not decide.** What is proven is written
  as a claim of its own; what stays open goes into the frontmatter's
  `uncertainty` and into the sentence with which it is handed over. Passing
  off a half delivery as `implemented` is the one change that nobody sees
  afterwards.
