# /// script
# requires-python = ">=3.12"
# dependencies = ["pypdf==6.16.2"]
# ///
"""The pdftotext seam: what the reference's pypdf reads from each PDF, in the form pdftotext prints.

Run: uv run --script docs/.superpowers/parity/stufe-4d-orakel/pypdf_pages.py
"""

import json
from pathlib import Path

from pypdf import PdfReader
from pypdf.errors import PyPdfError

WORLDS = Path(__file__).parents[4] / "testdata" / "cases" / "4d-worlds"
VERSION = "pdftotext version 25.07.0 (seam: the reference reads with pypdf 6.16.2)\nThe Poppler Developers\n"


def answer(pdf: Path) -> dict[str, object]:
    prefix = f"pdftotext -layout -enc UTF-8 -eol unix {pdf.name} -"
    try:
        pages = [page.extract_text(extraction_mode="layout") or "" for page in PdfReader(pdf).pages]
    except PyPdfError:
        return {"prefix": prefix, "exit": 1, "stdout": ""}
    return {"prefix": prefix, "exit": 0, "stdout": "".join(page + "\f" for page in pages)}


def main() -> None:
    for world in sorted(WORLDS.iterdir()):
        pdfs = sorted(world.rglob("*.pdf"))
        if not pdfs:
            continue
        answers = [{"prefix": "pdftotext -v", "exit": 0, "stdout": VERSION}, *(answer(pdf) for pdf in pdfs)]
        (world / "faketool.json").write_text(json.dumps({"answers": answers}, ensure_ascii=False, indent=1) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
