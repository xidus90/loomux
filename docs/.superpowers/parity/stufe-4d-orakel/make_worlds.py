# /// script
# requires-python = ">=3.12"
# ///
"""Builds testdata/cases/4d-worlds: one world per recorded convert case.

Run: uv run --script docs/.superpowers/parity/stufe-4d-orakel/make_worlds.py
"""

import json
import shutil
from pathlib import Path

ROOT = Path(__file__).parents[4]
WORLDS = ROOT / "testdata" / "cases" / "4d-worlds"
PDFS = ROOT / "testdata" / "convert" / "pdf"

REGISTRY = '[[area]]\nscope = "knowledge"\npath = "{{WORLD}}/vault"\n'
SECOND = '\n[[area]]\nscope = "project/x"\npath = "{{WORLD}}/x"\n'
MANIFEST = '[area]\nscope = "knowledge"\n\n[layout]\ninbox = "00 Eingang"\n'
MODEL = '[model]\nenabled = true\nendpoint = "http://127.0.0.1:11435"\n'
HALLO = "[00:00] Hallo zusammen.\n"
GOOD = "Der Bericht beschreibt die Abnahme der zweiten Scheibe."
# What a first run with describe on wrote for HALLO, on a day long past: the
# staged world keeps no modification time, so retrieved: is the run's day and
# both sides write the head again -- with the standing sentence, unasked.
FIRST_RUN = (
    "---\nsource_url:\nretrieved: 2020-01-01\nconverter: brain-transcript/1\nasr: true\n"
    f"description: {GOOD}\n---\n\n{HALLO}"
)


def text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8", newline="\n")


def world(name, files, *, registry=REGISTRY, manifest=MANIFEST, state=None, fixture=None, extra=None):
    """files maps an inbox name to text, to bytes, or to a PDF under testdata/convert/pdf."""
    root = WORLDS / name
    shutil.rmtree(root, ignore_errors=True)
    inbox = root / "vault" / "00 Eingang"
    inbox.mkdir(parents=True)
    if registry is not None:
        text(root / "registry.toml", registry)
    if manifest is not None:
        text(root / "vault" / ".brain.toml", manifest)
    for file, content in files.items():
        if isinstance(content, Path):
            shutil.copyfile(content, inbox / file)
        elif isinstance(content, bytes):
            (inbox / file).write_bytes(content)
        else:
            text(inbox / file, content)
    if state is not None:
        text(root / "config.toml", state)
    if fixture is not None:
        text(root / "ollama-fixture.json", json.dumps(fixture, ensure_ascii=False) + "\n")
    for rel, content in (extra or {}).items():
        text(root / rel, content)


def main() -> None:
    world("transcript-bracket", {"video (mHSOsy_usAg).txt": "[00:00] Hallo zusammen, das hier ist mein\n\n[00:04] Second Brain mit 2000 Notizen.\n"})
    world("transcript-range-lead-in", {"export.txt": "Titel des Videos\n\n00:01:05 - 00:01:57\nHallo zusammen.\n\n01:02:03 - 01:02:44\nSpät im Video.\n"})
    world("crlf-transcript", {"video.txt": b"00:00:00 - 00:00:05\r\nHallo zusammen.\r\n\r\n00:00:06 - 00:00:09\r\nUnd weiter.\r\n"})
    world("mixed-case-order", {"B.txt": "[00:00] Zwei.\n", "a.txt": "[00:00] Eins.\n", "_z.txt": "[00:00] Null.\n"})
    world("same-stem", {"doku.pdf": PDFS / "text.pdf", "doku.txt": "[00:00] Aus dem Transkript.\n"})
    world("unsupported-and-good", {"notiz.txt": "Nur Prosa.\n", "video.txt": HALLO})
    world("hand-written-target", {"video.txt": HALLO, "video.txt.md": "---\ntitle: Meine Notiz\n---\n\nHandarbeit.\n"})
    world("broken-utf8-source", {"kaputt.txt": b"[00:00] Anfang ist sauber.\n" + b"x" * 8300 + b"\n" + b"\xff\xfe" * 50, "video.txt": HALLO})
    world("broken-utf8-target", {"video.txt": HALLO, "video.txt.md": b"\xff\xfe" * 8192})
    world("pdf-text", {"buch.pdf": PDFS / "text.pdf"})
    world("pdf-umlaut-name", {"Bericht März.pdf": PDFS / "Bericht März.pdf"})
    world("pdf-scan", {"scan.pdf": PDFS / "blank.pdf", "video.txt": HALLO})
    world("pdf-pageless", {"leer.pdf": PDFS / "pageless.pdf"})
    world("pdf-partial", {"buch.pdf": PDFS / "mixed.pdf"})
    world("pdf-corrupt", {"buch.pdf": PDFS / "corrupt.pdf", "video.txt": HALLO})
    world("no-inbox", {}, manifest='[area]\nscope = "knowledge"\n')
    # A readonly area's declaration lies in the state directory, where both
    # sides look for it (registry.manifest_path); under vault/ it would leave
    # the area without an inbox, and the case would pass for that reason.
    world("readonly-area", {"video.txt": HALLO}, registry=REGISTRY + "readonly = true\n",
          manifest=None, extra={"areas/knowledge/.brain.toml": MANIFEST})
    world("single-file", {"video.txt": HALLO})
    world("no-registry", {"video.txt": HALLO}, registry=None)
    world("broken-manifest", {"video.txt": HALLO}, registry=REGISTRY + SECOND,
          extra={"x/.brain.toml": '[area]\nscope = "project/x"\n\n[privacy]\nmode = "cloud"\n'})
    describe = MODEL + "roles = { describe = true }\n"
    world("describe-kept", {"video.txt": HALLO}, state=describe, fixture={"response": GOOD})
    world("describe-refused", {"video.txt": HALLO}, state=describe, fixture={"response": "The report describes the second slice."})
    world("second-run", {"video.txt": HALLO, "video.txt.md": FIRST_RUN}, state=describe,
          fixture={"response": "Das Video begrüßt die Zuschauer zum Auftakt der Reihe."})
    place = MODEL + "roles = { place = true }\n"
    world("place-suggested", {"video.txt": HALLO}, registry=REGISTRY + SECOND, state=place,
          fixture={"response": '{"scope": "project/x", "grund": "Es passt."}'}, extra={"x/.keep": ""})
    world("place-unknown-scope", {"video.txt": HALLO}, registry=REGISTRY + SECOND, state=place,
          fixture={"response": '{"scope": "project/erfunden", "grund": "Es passt."}'}, extra={"x/.keep": ""})
    world("model-unreachable", {"video.txt": HALLO}, state=MODEL.replace("11435", "11436"))
    world("model-off-in-area", {"video.txt": HALLO}, manifest=MANIFEST + "\n[model]\nenabled = false\n",
          state=MODEL, fixture={"response": GOOD})
    world("broken-model-block", {"video.txt": HALLO}, state="[model]\nenabled = 5\n")
    world("endpoint-off-loopback", {"video.txt": HALLO}, state='[model]\nenabled = true\nendpoint = "http://192.0.2.1:11434"\n')


if __name__ == "__main__":
    main()
