"""Writes testdata/cases/3c-worlds: one state directory per world, its registry
spelling every path through {{WORLD}}, the areas declared under the old names
the reference reads.

Call: uv run python worlds.py   (from this directory; it rewrites the worlds)
"""

import shutil
from pathlib import Path

ROOT = Path(__file__).resolve().parents[4] / "testdata" / "cases" / "3c-worlds"

SOURCE = (
    "sources:\n  - id: s1\n    resource: brain://x/y\n    doc_id: d1\n"
    "    content_hash: h1\n    revision: 1\n"
)


def page(title: str, page_type: str = "Topic", extra: str = "", body: str = "") -> str:
    return (
        f"---\ntitle: {title}\ndescription: d\ntype: {page_type}\n{extra}{SOURCE}"
        f"generated:\n  by: test\n---\n\n# {title}\n{body}"
    )


def write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8", newline="\n")


def bundle(world: Path, name: str, scope: str, declaration: str = "") -> None:
    """A clean bundle of two linked pages with catalog and log."""
    repo = world / name
    write(repo / ".brain.toml", f'[area]\nscope = "{scope}"\n{declaration}')
    wiki = repo / "wiki"
    write(wiki / "index.md", "# Katalog\n\n* [a](a.md) -- first\n* [b](b.md) -- second\n")
    write(wiki / "log.md", "# Log\n\n## 2026-09-01\n")
    write(wiki / "a.md", page("a", body="\n[b](b.md)\n"))
    write(wiki / "b.md", page("b", body="\n[a](a.md)\n"))


def entry(scope: str, name: str | None, wiki: str | None = "wiki", extra: str = "") -> str:
    text = f'[[area]]\nscope = "{scope}"\n'
    if name is not None:
        text += f'path = "{{{{WORLD}}}}/{name}"\n'
    if wiki is not None:
        text += f'wiki = "{{{{WORLD}}}}/{name}/{wiki}"\n'
    return text + extra + "\n"


def registry(world: Path, *entries: str) -> None:
    write(world / "registry.toml", "".join(entries))


def one_area(world: Path) -> None:
    bundle(world, "repo-a", "project/a")
    registry(world, entry("project/a", "repo-a"))
    write(world / "loose" / "broken.md", "---\ntype: [unclosed\n---\n\nText\n")
    write(world / "loose" / "folder.md" / "inside.txt", "a directory named like a page\n")


def findings(world: Path) -> None:
    bundle(world, "repo-a", "project/a")
    wiki = world / "repo-a" / "wiki"
    write(wiki / "index.md", "# Katalog\n\n* [a](a.md)\n* [b](b.md)\n* [c](c.md)\n")
    write(wiki / "c.md", "---\ntitle: c\ntype: Topic\nstale_after: 2020-01-01\n---\n\n"
          "# c\n\n[gone](gone.md) [out](../outside.md)\n")
    registry(world, entry("project/a", "repo-a"))


def federation(world: Path) -> None:
    bundle(world, "repo-k", "knowledge", '\n[layout]\nhub = "Hub"\n')
    write(world / "repo-k" / "Hub" / "x.md", "hub page\n")
    bundle(world, "repo-a", "project/a")
    bundle(world, "repo-x", "engineering/x")
    write(world / "repo-x" / "wiki" / "a.md",
          page("a", body="\n[b](b.md)\n").replace("brain://x/y", "brain://project/a/topics/y"))
    registry(world,
             entry("knowledge", "repo-k", extra="signpost = true\nshared = true\n"),
             entry("project/a", "repo-a"),
             entry("engineering/x", "repo-x", extra="shared = true\n"))


def no_registry(world: Path) -> None:
    write(world / "README.txt", "a state directory nobody registered anything in\n")


def refusals(world: Path) -> None:
    bundle(world, "repo-a", "project/a")
    write(world / "repo-n" / ".brain.toml", '[area]\nscope = "project/nowiki"\n')
    write(world / "repo-g" / ".brain.toml", '[area]\nscope = "project/gone"\n')
    registry(world,
             entry("project/a", "repo-a"),
             entry("project/nowiki", "repo-n", wiki=None),
             entry("project/gone", "repo-g"))


def partial(world: Path) -> None:
    write(world / "repo-a" / ".brain.toml", '[area]\nscope = "project/a"\n')
    write(world / "repo-a" / "wiki" / "index.md", "# Eigener Katalog\n")
    registry(world, entry("project/a", "repo-a"))


def types_empty(world: Path) -> None:
    write(world / "repo-n" / ".brain.toml", '[area]\nscope = "project/nowiki"\n')
    registry(world, entry("project/nowiki", "repo-n", wiki=None))


def types_three(world: Path) -> None:
    bundle(world, "repo-a", "project/a", '\n[wiki]\ntypes = ["Balancing Rule"]\n')
    write(world / "repo-a" / "wiki" / "rule.md", page("rule", "Balancing Rule"))
    write(world / "repo-a" / "wiki" / "old.md", page("old", "Design Decision"))
    bundle(world, "repo-b", "project/b")
    write(world / "repo-b" / "wiki" / "rule.md", page("rule", "Balancing Rule"))
    write(world / "repo-b" / "wiki" / "arch.md", page("arch", "Architecture"))
    bundle(world, "repo-c", "engineering/c")
    write(world / "repo-c" / "wiki" / "untyped.md", "---\ntitle: untyped\n---\n\n# untyped\n")
    write(world / "repo-c" / "wiki" / "topics" / "Upper.md", page("upper", "topic"))
    registry(world, entry("project/a", "repo-a"), entry("project/b", "repo-b"),
             entry("engineering/c", "repo-c"))


def retype(world: Path, done: bool = False) -> None:
    bundle(world, "repo-a", "project/a")
    wiki = world / "repo-a" / "wiki"
    renamed = "Decision" if done else "Design Decision"
    write(wiki / "topics" / "one.md", page("one", renamed))
    write(wiki / "topics" / "Two.md", page("two", renamed, body="\n[a](../a.md)\n"))
    write(wiki / "draft.md", page("draft", "Design Decision Draft"))
    if not done:
        write(wiki / "quoted.md", page("quoted", '"Design Decision"'))
    bundle(world, "repo-r", "project/ro")
    registry(world, entry("project/a", "repo-a"), entry("project/ro", "repo-r", extra="readonly = true\n"))


WORLDS = {
    "one-area": one_area,
    "findings": findings,
    "federation": federation,
    "no-registry": no_registry,
    "refusals": refusals,
    "partial": partial,
    "types-empty": types_empty,
    "types-three": types_three,
    "retype": retype,
    "retype-done": lambda world: retype(world, done=True),
}

if __name__ == "__main__":
    if ROOT.exists():
        shutil.rmtree(ROOT)
    for name, build in WORLDS.items():
        build(ROOT / name)
        print(name)
