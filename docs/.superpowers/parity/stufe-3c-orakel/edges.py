"""The edge fixtures of sweep_edges_test.go, run through `lint_bundle`: what the
reference says where the mutation round found the Go form untested."""

import sys
import tempfile
from datetime import datetime
from pathlib import Path

from brain.wiki.lint import lint_bundle
from brain.wiki.page import count_conflicts

NOW = datetime(2026, 9, 23, 12).astimezone()
FULL = "sources:\n  - id: s\n    resource: https://x.invalid/s\n    doc_id: d\n    content_hash: h\n    revision: 1\n"


def sweep(pages: dict[str, str], **ctx: object) -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        links = "# c\n"
        for name, body in pages.items():
            (root / name).write_text(body, encoding="utf-8", newline="")
            links += f"* [{name}]({name})\n"
        (root / "index.md").write_text(links, encoding="utf-8", newline="")
        for f in lint_bundle(root, 30, now=NOW, **ctx):
            sys.stdout.write(f"  {f.relative}:{f.rule}:{f.severity.value}: {f.message}\n")


print("parent link")
sweep({"a.md": "---\ntype: Topic\n" + FULL + "---\n[up](..)\n"})
print("open_conflicts false")
sweep({"a.md": "---\ntype: Topic\nopen_conflicts: false\n" + FULL + "---\n> [!conflict] one\n"})
print("stale today")
sweep({"a.md": "---\ntype: Topic\nstale_after: 2026-09-23\n" + FULL + "---\n"})
print("project page built with a commit")
sweep({"done.md": "---\ntype: Topic\nrealization: implemented\nimplemented_in: abc123\n" + FULL + "---\n"}, is_project=True)
print("fences")
for text in [
    "```\n> [!conflict] a\n```\n> [!conflict] b\n```\n> [!conflict] c\n```\n",
    "```\n> [!conflict] a\n```\n> [!conflict] b\n",
    "~~~\n> [!conflict] a\n",
]:
    print(" ", count_conflicts(text))
print("open_conflicts true, two boxes")
sweep({"a.md": "---\ntype: Topic\nopen_conflicts: true\n" + FULL + "---\n> [!conflict] one\n\n> [!conflict] two\n"})
