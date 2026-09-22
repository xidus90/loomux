# Goldens: internal/brain/maintenance/testdata/merge-package.golden.md
#          internal/brain/maintenance/testdata/merge-case.golden.toml
# Call:    from the ultra-brain checkout:
#          uv run python merge_oracle.py <testdata directory>
#          Needs git on PATH; the repository it builds lives in a temporary
#          directory. Both files are written in binary.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), imported from the checkout named in sys.path below, which
#          has to stand at that tag.
# Source:  appendix of the task 12 report (stage 3a), unchanged.

# /// script
# requires-python = ">=3.12"
# ///
"""Write the merge-case goldens from the reference itself.

    uv run python merge_oracle.py <testdata directory>

Produces `merge-case.golden.toml` (`write_case`) and `merge-package.golden.md`
(`_merge_evidence` plus `_segments` plus `render_package`). Both go to disk in
binary, because Python's stdout turns every newline into CRLF on Windows.

A real repository is built and a real `_merge_evidence` is run over it, so the
golden carries git's own answers -- the order `git log` reports subjects in and
the names `git diff --name-only` prints -- rather than a list this script made
up. The Go side builds the same repository and asks its own `mergeEvidence`.

The three commit subjects are chosen for what they decide: an upper-case and a
lower-case file name pin that the path sort folds no case, a subject carrying a
fenced run pins the fence the package has to grow around it, and an umlaut pins
the encoding on the way through git.
"""

import subprocess
import sys
import tempfile
from datetime import datetime, timedelta, timezone
from pathlib import Path

sys.path.insert(0, r"C:\Users\micro\Documents\#GIT\ultra-brain\src")

from brain.maintenance.case import Case, write_case  # noqa: E402
from brain.maintenance.merge_events import MergeEvent  # noqa: E402
from brain.maintenance.package import render_package  # noqa: E402
from brain.maintenance.reconcile import _merge_evidence, _segments  # noqa: E402

out = Path(sys.argv[1])
NL = chr(10)
CR = chr(13)

PAGE = (
    "---" + CR + NL
    + "type: note" + CR + NL
    + "realization: planned" + CR + NL
    + "---" + CR + NL
    + CR + NL
    + "  eingerueckt, mit einem einsamen " + CR + " darin" + NL
    + NL
    + "   \t  " + NL
    + NL
    + "zweiter Absatz" + NL
).encode("utf-8")

IDENTITY = [
    "-c", "user.name=Test",
    "-c", "user.email=test@example.invalid",
    "-c", "commit.gpgsign=false",
]


def git(root, *arguments):
    done = subprocess.run(
        ["git", *arguments], cwd=root, check=True, capture_output=True
    )
    return done.stdout.decode("utf-8").strip()


def commit(root, name, text, subject):
    (root / name).write_bytes(text.encode("utf-8"))
    git(root, "add", "--all")
    git(root, *IDENTITY, "commit", "-q", "-m", subject)


with tempfile.TemporaryDirectory() as tmp:
    repo = Path(tmp) / "repo"
    repo.mkdir()
    git(repo, "init", "-q")
    git(repo, "config", "core.autocrlf", "false")
    (repo / ".gitattributes").write_bytes(b"* -text" + NL.encode())
    commit(repo, "base.txt", "base" + NL, "the commit before the range")
    first = git(repo, "rev-parse", "HEAD")
    commit(repo, "Zeta.txt", "zeta" + NL, "füge Zeta hinzu")
    commit(repo, "alpha.txt", "alpha" + NL, "fix ```code``` fences")
    last = git(repo, "rev-parse", "HEAD")

    event = MergeEvent(
        repo=str(repo),
        first=first,
        last=last,
        branch="feature",
        at=datetime(2026, 9, 20, 8, 0, 0, tzinfo=timezone.utc),
    )
    evidence = _merge_evidence(event)
    assert evidence is not None, "the range could not be read"

    page = Path(tmp) / "a.md"
    page.write_bytes(PAGE)
    segments = _segments([], page, evidence)

    # A merge case as it comes out of `_land_case`: no sources at all, `due`
    # out of the state vocabulary, and `change` as the weight both producers
    # carry.
    case = Case(
        id="a-2026-09-20-abcd",
        area="project/a",
        target="docs/wiki/a.md",
        target_hash="sha256:cc",
        state="due",
        trigger="merge",
        weight="change",
        created=datetime(2026, 9, 20, 8, 0, 0, tzinfo=timezone(timedelta(hours=2))),
        sources=(),
    )
    (out / "merge-package.golden.md").write_bytes(
        render_package(case, segments).encode("utf-8")
    )
    written = Path(tmp) / "case.toml"
    write_case(written, case)
    (out / "merge-case.golden.toml").write_bytes(written.read_bytes())

print("blocks:", len(evidence), "segments:", len(segments))
