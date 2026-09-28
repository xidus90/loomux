"""Prints the reference's verdicts where a Go port of detect, transcript and header could part.

Run with the reference's interpreter, on the tag loomux-3-source:
  "C:/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" \
    docs/.superpowers/parity/stufe-4d-orakel/convert_detect.py
"""

import sys
import tempfile
from datetime import date
from pathlib import Path

from brain.convert.detect import Format, detect
from brain.convert.header import converted_by, description_of, header
from brain.convert.transcript import to_paragraphs

print(sys.version)
tmp = Path(tempfile.mkdtemp())


def detected(name: str, data: bytes) -> str:
    path = tmp / name
    path.write_bytes(data)
    return str(detect(path))


# Path.suffix of Python 3.14 strips the leading dots first.
for name in ["a.txt", ".txt", "a.", "a", "a.b.PDF", "..x", "a..", "x.txt.", ".", "..", "a. "]:
    print("suffix", repr(name), repr(Path(name).suffix))
print("detect ..txt", detected("..txt", b"[00:00] Hallo.\n"))
print("detect ..pdf", detected("..pdf", b"%PDF-1.7\n"))

# Python's \s in the range line.
for tail in ["\xa0", "\x0b", "\x1c", "\x85", "\u3000"]:
    text = "00:00:00 - 00:00:57" + tail + "\nHallo.\n"
    print("range tail", repr(tail), detected("r.txt", text.encode()),
          repr(to_paragraphs(text, Format.TRANSCRIPT_RANGE)))
print("body", repr(to_paragraphs("[00:00] a\x1fb\u3000c\xa0d\n", Format.TRANSCRIPT_BRACKET)))

# Python's \d takes every script's decimal digits.
text = "[\u0660\u0660:\u0660\u0665] Hallo.\n"
print("arabic-indic digits", detected("d.txt", text.encode()),
      repr(to_paragraphs(text, Format.TRANSCRIPT_BRACKET)))
# Between ASCII marks, such a mark is cut out as a mark: its text goes, and
# it becomes an anchor only where a paragraph starts at it.
text = "[00:00] Eins.\n[\U00000660\U00000660:\U00000660\U00000665] Zwei.\n[00:09] Drei.\n"
print("arabic-indic mark between", repr(to_paragraphs(text, Format.TRANSCRIPT_BRACKET)),
      repr(to_paragraphs(text, Format.TRANSCRIPT_BRACKET, threshold=1)))

# CRLF is one character in the head: the mark stands at character 6000.
print("crlf-far", detected("crlf-far.txt", ("a\r\n" * 3000 + "[00:00] Hallo.\r\n").encode()))

# TextIOWrapper decodes whole chunks: "[00:00] " and 4092 "ä" make the first
# chunk of 8192 bytes, 4092 "x" complete the 8192 characters, and the broken
# byte stands ten characters after them, inside the second chunk.
head = "[00:00] " + "\u00e4" * 4092
data = head.encode() + b"x" * (4092 + 10) + b"\xff" + b"y" * 100
print("broken byte after the head, inside the chunk", detected("chunk.txt", data))

# The second chunk is scaled by the first one's bytes per character: 8 ASCII
# bytes and 2046 four-byte characters make 8192 bytes for 2054 characters,
# so the second read asks for int(8192 / 2054 * 6138) = 24480 bytes. With
# fixed chunks of 8192 bytes the byte at 8192 + 10000 is never read; within
# the scaled chunk it is, and one just past the chunk is not.
head = ("[00:00] " + "\U0001f600" * 2046).encode()
for offset in [10000, 24479, 24480]:
    data = head + b"x" * offset + b"\xff" + b"y" * 100
    print("broken byte at 8192 +", offset, detected(f"scaled{offset}.txt", data))

# Python's \s and \S in the head lines.
for label, text in [
    ("description nbsp before", "---\ndescription:\xa0Satz.\n---\n"),
    ("description nbsp after", "---\ndescription: Satz.\xa0\n---\n"),
    ("description ideographic after", "---\ndescription: Satz.\u3000\n---\n"),
    ("description crosses a line", "---\ndescription:\nsource_url: x\n---\n"),
]:
    print(label, repr(description_of(text)))
for label, text in [
    ("converter ideographic before", "---\nconverter:\u3000brain-pdf/1\n---\n"),
    ("converter nbsp after", "---\nconverter: brain-pdf/1\xa0\n---\n"),
    ("converter crosses a line", "---\nconverter:\nfoo\n---\n"),
]:
    print(label, repr(converted_by(text)))

# The whole head, four lines and five. An empty string is written as an
# empty line, unlike None; no caller passes one.
print("head four", repr(header(source_url="", retrieved=date(2026, 8, 24), converter="brain-pdf/2", asr=False)))
print("head five", repr(header(source_url="", retrieved=date(2026, 9, 7), converter="pdf", asr=False,
                               description="Der Bericht beschreibt die Abnahme.")))
print("head empty description", repr(header(source_url="", retrieved=date(2026, 9, 7), converter="pdf",
                                            asr=False, description="")))
