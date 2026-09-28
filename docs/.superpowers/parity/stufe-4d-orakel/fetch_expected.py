"""What the reference's fetch writes for the yt-dlp recording of Task 1.

The track is chosen as `_run_yt_dlp` chooses it (fetch.py:118-124); the
download itself is replaced by the recorded file, which is the one network
seam the reference's tests do not reach either. Within a language the
recorded file is yt-dlp's choice, the last json3 track listed, where the
reference downloaded the first; the spec releases that difference (row
"fetch: Spur einer Sprache"), and this oracle does not see it.

Run with the reference's interpreter, on the tag loomux-3-source:
  "C:/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" \
    docs/.superpowers/parity/stufe-4d-orakel/fetch_expected.py
"""

import json
import shutil
from pathlib import Path

from brain.convert.fetch import fetch, to_inbox

RECORDING = Path(__file__).parents[4] / "testdata" / "convert" / "ytdlp" / "mHSOsy_usAg"
URL = "https://www.youtube.com/watch?v=mHSOsy_usAg"


def runner(url: str) -> tuple[dict[str, str], str]:
    files = RECORDING / "files"
    info = json.loads((files / "v.info.json").read_text(encoding="utf-8"))
    meta = {"title": info.get("title") or "", "upload_date": "", "channel": ""}
    for key in ("subtitles", "automatic_captions"):
        by_lang = info.get(key) or {}
        lang = "de" if by_lang.get("de") else "en"
        tracks = by_lang.get("de") or by_lang.get("en") or []
        if any(track.get("ext") == "json3" for track in tracks):
            return meta, (files / f"v.{lang}.json3").read_text(encoding="utf-8")
    return meta, ""


def main() -> None:
    out = RECORDING / "expected"
    shutil.rmtree(out, ignore_errors=True)
    out.mkdir()
    written = to_inbox(fetch(URL, runner=runner), URL, out)
    (out / "name").write_text(written.name + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
