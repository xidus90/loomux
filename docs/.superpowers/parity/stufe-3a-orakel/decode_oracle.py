# Golden:  none. It prints the answers that the table of
#          TestDecodeReplacesEachMaximalSubpart (internal/brain/maintenance)
#          was checked against: CPython's `decode("utf-8", errors="replace")`
#          after the CRLF fold, for the sixteen inputs below.
# Call:    PYTHONIOENCODING=utf-8 uv run --script decode_oracle.py
#          (on Windows the output otherwise fails on cp1252).
# Against: CPython itself, not ultra-brain; the rule it checks is the one
#          `content_hash` applies at ultra-brain tag `loomux-3-source` (3cc72d2).
# Source:  appendix of the task 10 report (stage 3a), unchanged. The rows the
#          mutant round added to that table later were checked against CPython
#          by other means and are not in this script.

# /// script
# requires-python = ">=3.12"
# ///
CASES = [
    b"\xff\xff", b"\xe2\x82", b"\xe2\x82A", b"\xe0\x80\x80", b"\xed\xa0\x80",
    b"\xc0\xaf", b"\xf5\x80", b"\xc2", b"\xf0\x9f\x92", b"\xf1\x80A",
    b"\xf4\x90", b"\xeeA", b"\x80", b"\xc3\xa4\xffb", b"a\r\n\xffb",
    b"\xe2\x82\xac",
]
for raw in CASES:
    print(repr(raw), "->", repr(raw.replace(b"\r\n", b"\n").decode("utf-8", errors="replace")))
