"""Builds one wiki world with the sweep's edge cases and prints what
`lint_bundle` finds in it, one line per finding, for a byte comparison."""

import os
import shutil
import sys
from datetime import datetime
from pathlib import Path

from brain.wiki.lint import lint_bundle

base = Path(sys.argv[1])
if base.exists():
    shutil.rmtree(base)
wiki = base / "wiki"
other_wiki = base / "other" / "wiki"
other_wiki.mkdir(parents=True)
hub = base / "Hub"
hub.mkdir(parents=True)
(hub / "q.md").write_text("hub\n", encoding="utf-8")

SRC = "sources:\n  - id: s1\n    resource: raw/s.md\n    doc_id: d\n    content_hash: h\n    revision: 1\n"


def w(rel, text, newline="\n"):
    p = wiki / rel
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8", newline=newline)
    return p


w("raw/s.md", "raw source, no frontmatter\n")
w("index.md", "# Catalog\n\n* [a](a.md)\n* [b](topics/b.md)\n* [c](sub%20dir/c.md)\n"
  "* [out](../other/wiki/x.md)\n* [ext](https://example.invalid/)\n* [bad](%zz.md)\n* [[Upper]]\n")
w("_schema.md", "---\ntype: Topic\n---\n[dead](nowhere.md)\n")
w("a.md", "---\ntitle: a\ntype: Topic\n" + SRC + "---\n"
  "[b](topics/b.md) [miss](missing.md) [abs](/abs.md) [up](../up.md) [q](c.md?x=1#f)\n"
  "[pct](%zz.md) ![img](i.md) [[Wiki Link]] [[ spaced | alias]] [dup](topics/b.md)\n"
  "[root](/a.md) [climb](/../escape.md) [scheme](c:/x.md) [frag](#only)\n")
w("topics/b.md", "---\ntype: Design Decision\nsources:\n  - id: s2\n    resource: ../raw/s.md\n"
  "  - resource: r\n    doc_id: d\n---\n[a](../a.md)\n")
w("topics/index.md", "# nested\n[o](orphaned.md)\n")
w("topics/orphaned.md", "---\ntype: topic\n---\norphan\n")
w("sub dir/c.md", "---\ntype: [unclosed\n---\n> [!conflict] one\n")
w("d.md", "---\ntype: ''\nopen_conflicts: 2\nstale_after: 2020-01-01\nrealization: implemented\n" + SRC
  + "---\n```\n> [!CONFLICT] fenced\n```\n  > [!conflict] outside\n[a](a.md)\n", newline="\r\n")
e = w("e.md", "---\ntype: Decision\nrealization: planned\nimplemented_in: ''\nsources:\n"
      "  - id: s3\n    resource: brain://project/x/topics/y\n    doc_id: d\n    content_hash: h\n    revision: 2\n"
      "  - id: s4\n    resource: brain://engineering/x/z\n    doc_id: d\n    content_hash: h\n    revision: 2\n"
      "  - id: s5\n    resource: brain://\n    doc_id: d\n    content_hash: h\n    revision: 2\n---\n[a](a.md)\n")
old = datetime(2026, 1, 1, 12).timestamp()
os.utime(e, (old, old))
w("f.md", "---\n- a\n- b\n---\n[a](a.md)\n")
w("Upper.md", "---\ntype: Balancing Rule\nopen_conflicts: true\n" + SRC + "---\n> [!conflict] x\n")
w("g.md", "---\ntype: Metric\nstale_after: '2020-01-01'\nsources: nope\n---\n[a](a.md)\n")
w("h.md", "---\n---\n")
(wiki / "i.md").write_bytes(b"---\ntype: Topic\n---\nbad \xff byte [a](a.md)\n")

now = datetime(2026, 9, 23, 12).astimezone()
found = lint_bundle(
    wiki, 30, now=now,
    expected_targets=(
        ("project/p", (other_wiki.resolve(),)),
        ("project/q", (other_wiki.resolve() / "nope", (hub / "q.md").resolve())),
        ("project/r", ((base / "rwiki").resolve(),)),
    ),
    is_shared=True, shared_scopes=frozenset({"engineering/x"}),
    declared_types=frozenset({"Balancing Rule"}), is_project=True,
)
for f in found:
    sys.stdout.buffer.write(f"{f.relative}:{f.rule}:{f.severity.value}: {f.message}\n".encode("utf-8"))
