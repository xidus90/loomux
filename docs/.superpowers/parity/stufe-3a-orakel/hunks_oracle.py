# Golden:  internal/brain/maintenance/testdata/hunks.golden.json
# Call:    from the ultra-brain checkout, so `brain` and its dependencies
#          import:  uv run python hunks_oracle.py <golden file>
#          The file is written in binary.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), imported from the checkout named in sys.path below, which
#          has to stand at that tag.
# Source:  the scratchpad copy that wrote the committed golden. The appendix
#          of the task 11 report holds an earlier state of this script with
#          nineteen vectors; the twentieth, "the first of two equal blocks",
#          came with fix round 1 and is only here. The twenty names match the
#          keys of the golden and of `goldenHunks()` in diff_internal_test.go.

# /// script
# requires-python = ">=3.12"
# ///
"""Write testdata/hunks.golden.json from the reference `_hunks`.

Run from ultra-brain so `brain` and its dependencies are importable:

    uv run --project C:/Users/micro/Documents/#GIT/ultra-brain python hunks_oracle.py <out>

The output goes to the file in binary, never through stdout: Python's stdout
turns every `\n` into `\r\n` on Windows, which would be a wrong golden.
"""

import json
import sys

sys.path.insert(0, r"C:\Users\micro\Documents\#GIT\ultra-brain\src")

from brain.maintenance.reconcile import _Changed, _hunks  # noqa: E402


def changed(relative, text, baseline):
    return _Changed(
        doc_id="d",
        relative=relative,
        revision=1,
        content_hash="sha256:00",
        text=text,
        baseline=baseline,
    )


NL = chr(10)
TEN = "".join(f"line {i}\n" for i in range(1, 11))
TWENTY = "".join(f"line {i}\n" for i in range(1, 21))
LONG = "".join(f"line {i}\n" for i in range(1, 251))

vectors = {
    # One changed line in the middle: one hunk with three lines of context.
    "middle": changed("src/a.go", TEN.replace("line 5\n", "line five\n"), TEN),
    # Two changes far apart: the grouping splits them into two hunks.
    "two hunks": changed(
        "src/a.go",
        TWENTY.replace("line 2\n", "line two\n").replace("line 18\n", "line eighteen\n"),
        TWENTY,
    ),
    # The new text has no trailing newline, so the marker applies to its last
    # line -- and to nothing else.
    "no newline": changed("src/a.go", "a\nb\nc", "a\nb\nc\n"),
    # The old text has none either: the marker lands on a removed line too.
    "no newline before": changed("src/a.go", "a\nb\nc\n", "a\nb\nc"),
    # No verified baseline: `/dev/null`, the note above the header, and the
    # whole new file as one hunk.
    "no baseline": changed("src/a.go", "a\nb\n", None),
    # A committed empty file is a baseline that happens to be empty: the
    # header names the file and not `/dev/null`.
    "empty baseline": changed("src/a.go", "a\nb\n", ""),
    # The new text is empty, the old one was not.
    "emptied": changed("src/a.go", "", "a\nb\n"),
    # An added empty line is a `+` on its own, and it keeps the newline that
    # belongs to it -- `removesuffix`, not `rstrip`.
    "added empty line": changed("src/a.go", "a\n\nb\n", "a\nb\n"),
    # A lone carriage return stays inside a line: `_lines` splits at `\n` and
    # nowhere else.
    "lone carriage return": changed("src/a.go", "a\rb\nc\n", "a\nc\n"),
    # Past the 200-line mark where autojunk starts, with a line repeated often
    # enough to be dropped as popular.
    "autojunk": changed(
        "src/a.go",
        LONG.replace("line 7\n", "\n").replace("line 200\n", "\n") + "\n" * 20 + "tail\n",
        LONG + "\n" * 20,
    ),
    # Nothing moved: `unified_diff` yields nothing at all, so there is no
    # header to take `lines[:2]` from.
    "identical": changed("src/a.go", TEN, TEN),
    # A path with a space and a non-ASCII name, both of which go into the
    # header verbatim.
    "umlaut path": changed("src/ä b.md", "neu\n", "alt\n"),
    # Both sides empty: `unified_diff` has no opcode at all, and `difflib`
    # invents one so its two fixups have something to index.
    "both empty": changed("src/a.go", "", ""),
    # The case autojunk decides on its own. Every second line of b is the same
    # one, 100 times over 200 lines, so it counts as popular and is struck
    # from the index; the matcher then finds no anchor at all and the whole
    # file becomes one replacement. Without the rule the same input yields 200
    # opcodes and a hunk per repeated line.
    "autojunk strikes a popular line": changed(
        "src/a.go",
        "".join(f"v{i}\nx\n" for i in range(100)),
        "".join(f"u{i}\nx\n" for i in range(100)),
    ),
    # A struck line in front of the anchor: the matcher finds a one-line block
    # on a unique line and then grows it outwards over the popular ones, which
    # it may because a struck line is not junk.
    "a struck line grows the block": changed(
        "src/a.go",
        "".join(f"x\nu{i}\n" for i in range(100)).replace("u50\n", "CHANGED\n"),
        "".join(f"x\nu{i}\n" for i in range(100)),
    ),
    # A line repeated before and after the longest block, so the recursion
    # into the box on its left meets an occurrence beyond that box's end and
    # the one into the box on its right meets occurrences before its start.
    "a repeated line on both sides": changed(
        "src/a.go",
        "m\nA\nm\np\nq\nr\ns\nt\nm\nZ\nm\n",
        "m\na\nm\np\nq\nr\ns\nt\nm\nz\nm\n",
    ),
    # Below the 200-line mark autojunk does not start at all, however popular
    # a line is: 150 lines, every second one the same, and the matcher still
    # anchors on it.
    "autojunk stays off below the mark": changed(
        "src/a.go",
        "".join(f"v{i}" + NL + "x" + NL for i in range(75)),
        "".join(f"u{i}" + NL + "x" + NL for i in range(75)),
    ),
    # Exactly `len(b) // 100 + 1` occurrences, which is one too few to be
    # popular: over 200 lines a line appearing three times stays in the index,
    # and it is the only line the two sides still share.
    "three occurrences are not popular": changed(
        "src/a.go",
        "".join(("x" if i in (0, 100, 199) else f"v{i}") + NL for i in range(200)),
        "".join(("x" if i in (0, 100, 199) else f"u{i}") + NL for i in range(200)),
    ),
    # Five unchanged lines between two changes: fewer than the six that would
    # end the group, so the two stay in one hunk.
    "one hunk over a short stretch": changed(
        "src/a.go",
        TWENTY.replace("line 2" + NL, "line two" + NL).replace("line 8" + NL, "line eight" + NL),
        TWENTY,
    ),
    # Three lines against one, and the one is repeated: the matcher has to
    # take the FIRST of the two equally long blocks, so the kept line is the
    # leading one and the two added ones follow. `>=` in place of `>` takes
    # the later block and puts the kept line at the end.
    "the first of two equal blocks": changed("src/a.go", "L1" + NL + "L0" + NL + "L1" + NL, "L1" + NL),
}

out = {name: _hunks(item) for name, item in vectors.items()}
with open(sys.argv[1], "wb") as handle:
    handle.write(json.dumps(out, ensure_ascii=False, indent=1, sort_keys=True).encode("utf-8"))
    handle.write(b"\n")
