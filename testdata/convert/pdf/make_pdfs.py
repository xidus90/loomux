# /// script
# requires-python = ">=3.12"
# dependencies = ["pypdf==6.16.2"]
# ///
"""The PDFs stage 4d measures and tests with, built by hand as the reference's tests build them.

Run: uv run --script testdata/convert/pdf/make_pdfs.py
"""

from pathlib import Path

from pypdf import PdfReader, PdfWriter

LONG_TEXT = (
    "Hallo aus dem Pruefbestand. Diese Zeile testet die Extraktion aus einer "
    "von Hand geschriebenen PDF-Datei, ohne Texterkennung und ohne fremde "
    "Erzeugerbibliothek im Testbestand. Der Inhalt bleibt frei erfunden und "
    "dient allein dazu, die Schwelle von hundert Zeichen je Seite sicher zu "
    "ueberschreiten, damit die Datei nicht faelschlich als Scan gilt und der "
    "Test etwas Sinnvolles pruefen kann."
)


def assemble(objects: list[bytes]) -> bytes:
    out = bytearray(b"%PDF-1.4\n")
    offsets: list[int] = []
    for number, body in enumerate(objects, start=1):
        offsets.append(len(out))
        out += f"{number} 0 obj\n".encode() + body + b"\nendobj\n"
    table = len(out)
    out += f"xref\n0 {len(objects) + 1}\n".encode() + b"0000000000 65535 f \n"
    for offset in offsets:
        out += f"{offset:010d} 00000 n \n".encode()
    out += (
        f"trailer\n<< /Size {len(objects) + 1} /Root 1 0 R >>\nstartxref\n{table}\n%%EOF\n"
    ).encode()
    return bytes(out)


def pages(texts: list[str]) -> bytes:
    n = len(texts)
    font = 3 + 2 * n
    kids = " ".join(f"{3 + 2 * i} 0 R" for i in range(n))
    objects: list[bytes] = [
        b"<< /Type /Catalog /Pages 2 0 R >>",
        f"<< /Type /Pages /Kids [{kids}] /Count {n} >>".encode(),
    ]
    for text in texts:
        stream = f"BT /F1 12 Tf 72 720 Td ({text}) Tj ET".encode("latin-1")
        # pdftotext clips at the page edge, pypdf does not: 5000 pt holds the 390/783-char lines for both.
        objects.append(
            f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 5000 842] "
            f"/Resources << /Font << /F1 {font} 0 R >> >> "
            f"/Contents {len(objects) + 2} 0 R >>".encode()
        )
        objects.append(
            b"<< /Length " + str(len(stream)).encode() + b" >>\nstream\n" + stream + b"\nendstream"
        )
    objects.append(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
    return assemble(objects)


def main() -> None:
    here = Path(__file__).parent
    files = {
        "text.pdf": pages([LONG_TEXT]),
        "paragraphs.pdf": pages([LONG_TEXT + "\\n\\n\\n" + LONG_TEXT]),
        "blank.pdf": pages([""]),
        "pageless.pdf": assemble(
            [b"<< /Type /Catalog /Pages 2 0 R >>", b"<< /Type /Pages /Kids [] /Count 0 >>"]
        ),
        "mixed.pdf": pages([LONG_TEXT, LONG_TEXT, LONG_TEXT, *([""] * 7)]),
        "allscan.pdf": pages(["", ""]),
        "corrupt.pdf": b"%PDF-1.4\n",
        "Bericht März.pdf": pages([LONG_TEXT]),
    }
    for name, data in files.items():
        (here / name).write_bytes(data)
    writer = PdfWriter(clone_from=PdfReader(here / "text.pdf"))
    # RC4 because pypdf writes it without an extra package; AES needs one.
    writer.encrypt(user_password="geheim", owner_password="geheim", algorithm="RC4-128")
    with (here / "encrypted.pdf").open("wb") as out:
        writer.write(out)


if __name__ == "__main__":
    main()
