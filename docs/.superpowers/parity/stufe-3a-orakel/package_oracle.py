# Golden:  internal/brain/maintenance/testdata/package.golden.md
# Call:    uv run --script package_oracle.py
#          It writes `package.golden.txt` into the current directory, in
#          binary; that file is copied over package.golden.md.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), imported from the checkout named in sys.path below, which
#          has to stand at that tag.
# Source:  appendix of the task 9 report (stage 3a), unchanged. Its inputs are
#          those of `goldenSegments()` in package_test.go.

# /// script
# requires-python = ">=3.12"
# ///
import sys
from datetime import datetime, timezone, timedelta

sys.path.insert(0, r"C:\Users\micro\Documents\#GIT\ultra-brain\src")

from brain.maintenance.case import Case
from brain.maintenance.package import build_package, render_package

segments = build_package(
    diff_hunks=[
        "package a\n",
        "--- a/x\n+++ b/x\n```\nfenced\n```\ntail",
        "a\rb\n```` run\nc",
        "\u00e4" * 85 + "\nsecond line",
    ],
    page_paragraphs=["  leading and trailing  \n\nrest"],
    sources=[("doc-1", "src/" + "z" * 100 + ".md"), ("doc-2", "src/b.md")],
)
case = Case(
    id="loomux-2026-09-19-abcd",
    area="project/loomux",
    target="docs/wiki/a.md",
    target_hash="sha256:aa",
    state="source_changed",
    trigger="source_change",
    weight="change",
    created=datetime(2026, 9, 19, 10, 0, 0, tzinfo=timezone(timedelta(hours=2))),
    sources=(),
)
open("package.golden.txt", "wb").write(render_package(case, segments).encode("utf-8"))
