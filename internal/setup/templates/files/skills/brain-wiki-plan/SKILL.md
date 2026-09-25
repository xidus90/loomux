---
name: brain-wiki-plan
description: After finished work, determine which wiki pages have to be pulled along. Applies to "what has to go into the wiki", "which pages does this affect", "wiki plan", "the work is done, what is missing in the wiki" — before the merge, as a survey. Not for the writing itself; brain-ingest does that, and after the merge brain-land.
---

# Wiki plan: what the finished work owes the wiki

This procedure runs **before the merge**, when the work stands. It
establishes which pages are no longer right — and stops exactly there.
**Nothing is written**, neither into the wiki nor into the review centre. The
result is a list, and what becomes of it arises later by itself: the merge to
`main` opens a case for every affected page, which `brain-land` works on.

## Why it writes nothing

Before the merge nothing is true yet. A page that reports `implemented` now
claims something about a commit that does not exist on `main` — and
`house/implemented-without-commit` only catches that afterwards. The plan is
therefore a survey, not an anticipation: it says what will have to be done,
and leaves the doing to the moment at which it is right.

The second half of the same reason: the merge trigger forms the cases anyway.
A page created here by hand would then stand beside a case about the same
page, and one of the two would be wrong.

## Step 0 — name the area

As with every procedure over the stocks: **if the area is not named, ask.**
`loomux brain catalog` lists the available areas. The wiki plan applies to
project areas; `realization` is a project field, and in a shared area the lint
does not check it at all.

## Steps 1 to 5 — the procedure

1. **Establish what the work was.** The commit range since branching off
   `main`, with paths and commit messages. **No source text** — the evidence
   of this trigger stays with paths and subject lines, and what is read here
   determines what may later land in a package.
2. **Find the affected pages through the wiki catalog.** Read the bundle's
   `index.md`, do not search the bundle. The same cost reason as with
   `brain-ingest`: the catalog is what keeps this procedure affordable over a
   large stock.
3. **Open exactly these pages — never the whole wiki.** Per page the
   frontmatter and the paragraphs that carry a promise.
4. **Sort each page** — see the three bins below.
5. **Present the list.** Page, bin, one sentence why. And explicitly what is
   **not** affected, when the work seemed to have touched a page and yet does
   not.

## Step 4 — the three bins

Exactly three possibilities for every page:

- **It is delivered.** The page planned something, the work built it. After
  the merge `realization: implemented` and `implemented_in` become due — that
  is `brain-land`.
- **It is outdated.** The wording describes something that is different now.
  The paragraph is named, not rewritten yet.
- **It is missing.** The work created something for which there is no page.
  It comes into being through `brain-ingest`, with the work as its source.

**When unclear, mark, do not decide.** A page for which it is open whether
the work delivers it goes into the list with that uncertainty. Anticipating
would be especially expensive here: `implemented` is a promise, and taking a
promise back costs more than giving it one round later.

## What the check already says about it

```bash
loomux brain check bundle --scope <area>
```

Two findings belong to this procedure and are a good entry into step 2:

| Finding | means |
|---|---|
| `house/long-planned` | `planned` for a long time and never touched — is anyone still planning this? |
| `house/implemented-without-commit` | `implemented` without `implemented_in`, so an unchecked promise |

Both are hints at candidates, no substitute for step 1: the check only knows
the pages, not the work.
