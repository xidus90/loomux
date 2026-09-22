# Goldens: internal/brain/maintenance/testdata/case-package.golden.md
#          internal/brain/maintenance/testdata/case.golden.toml
# Call:    from the ultra-brain checkout:
#          uv run python case_oracle.py <testdata directory>
#          Both files are written in binary.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), imported from the checkout named in sys.path below, which
#          has to stand at that tag.
# Source:  appendix of the task 11 report (stage 3a), unchanged.

# /// script
# requires-python = ">=3.12"
# ///
"""Write the case and package goldens from the reference itself.

    uv run python case_oracle.py <testdata directory>

Produces `case.golden.toml` (`write_case`) and `case-package.golden.md`
(`_segments` plus `render_package`). Both go to disk in binary, because
Python's stdout turns every newline into CRLF on Windows.

The page is written as bytes, deliberately mixed: CRLF in the frontmatter, a
lone CR inside a paragraph, an indented paragraph, and a paragraph of nothing
but whitespace. Those are the four things `_body`, `_segments` and Python's
text-mode read decide between them.
"""

import sys
import tempfile
from datetime import datetime, timedelta, timezone
from pathlib import Path

sys.path.insert(0, r"C:\Users\micro\Documents\#GIT\ultra-brain\src")

from brain.maintenance.case import Case, SourceState, write_case  # noqa: E402
from brain.maintenance.package import render_package  # noqa: E402
from brain.maintenance.reconcile import _Changed, _segments  # noqa: E402

out = Path(sys.argv[1])
NL = chr(10)
CR = chr(13)

PAGE = (
    "---" + CR + NL
    + "type: note" + CR + NL
    + "sources:" + CR + NL
    + "  - doc_id: d1" + CR + NL
    + "---" + CR + NL
    + CR + NL
    + "  eingerueckt, mit einem einsamen " + CR + " darin" + NL
    + NL
    + "   \t  " + NL
    + NL
    + "zweiter Absatz" + NL
).encode("utf-8")

# Out of doc-id order on purpose: `_segments` sorts, and a port that took the
# slice as it came would render D and Q the other way round.
SOURCES = [
    _Changed(doc_id="d2", relative="src/zwei.go", revision=2, content_hash="sha256:bb",
             text="zwei" + NL, baseline="eins" + NL),
    _Changed(doc_id="d1", relative="src/ä eins.go", revision=1, content_hash="sha256:aa",
             text="neu" + NL, baseline=None),
]

# Every optional field set, and `note` carries the characters `_quote` exists
# for: a newline, a quotation mark, a backslash, a tab and a DEL.
CASE = Case(
    id="a-2026-09-20-abcd",
    area="project/a",
    target="docs/wiki/a.md",
    target_hash="sha256:cc",
    state="source_changed",
    trigger="source_change",
    weight="change",
    created=datetime(2026, 9, 20, 8, 0, 0, tzinfo=timezone(timedelta(hours=2))),
    sources=(
        SourceState(doc_id="d1", revision=1, content_hash="sha256:aa"),
        SourceState(doc_id="d2", revision=2, content_hash="sha256:bb"),
    ),
    note='zwei' + NL + 'zeilen, ein "Zitat", ein \\ und ein \t sowie \x7f',
    superseded_proposal="superseded-proposal.md",
    manual=True,
    local_only=True,
    prompt_version="propose/7",
)

with tempfile.TemporaryDirectory() as tmp:
    page = Path(tmp) / "a.md"
    page.write_bytes(PAGE)
    segments = _segments(SOURCES, page)
    (out / "case-package.golden.md").write_bytes(render_package(CASE, segments).encode("utf-8"))
    written = Path(tmp) / "case.toml"
    write_case(written, CASE)
    (out / "case.golden.toml").write_bytes(written.read_bytes())
print("segments:", len(segments))
