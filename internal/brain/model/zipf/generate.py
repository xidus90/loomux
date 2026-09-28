# /// script
# requires-python = ">=3.12"
# dependencies = ["wordfreq==3.1.1"]
# ///
"""Writes de.txt.gz, the German Zipf frequencies chopped_words asks about, and NOTICE.md.

Three sections. [common]: every key without a digit run whose Zipf
frequency is at least 3.0. [mid]: every such key at 2.5 or more and below
3.0, of any length -- the judge counts a part's length as written, and
casefolding makes `Grüße` (five) the key `grüsse` (six). [digits]: every key
with a digit run and its raw frequency, because wordfreq multiplies that by
digit_freq of the token the caller asked about, which no table of Zipf
numbers can hold.

NOTICE.md carries the license and wordfreq's own account of its sources,
copied from the installed package's README so that no word is lost.

Run: uv run --script internal/brain/model/zipf/generate.py
"""

import gzip
from importlib.metadata import distribution
from pathlib import Path

from wordfreq import get_frequency_dict, zipf_frequency
from wordfreq.numbers import MULTI_DIGIT_RE

HEAD = """# Third-party notice: German word frequencies

loomux embeds `internal/brain/model/zipf/de.txt.gz`, a table derived from the
German word frequencies of wordfreq 3.1.1 by Robyn Speer
(<https://github.com/rspeer/wordfreq>), Copyright 2022 Robyn Speer. It holds
the Zipf frequency class of every word the chopped-word judge asks about, and
the raw frequency of the keys that carry digits.

The table is an adaptation of wordfreq's data and is licensed under the
Creative Commons Attribution-ShareAlike 4.0 International license
(CC BY-SA 4.0, <https://creativecommons.org/licenses/by-sa/4.0/>), as the data
it derives from. It is not covered by loomux's own license. `generate.py` in
the same directory rebuilds it.

What follows is quoted from wordfreq's README, version 3.1.1.

"""


def section(readme: str, title: str) -> str:
    """One `## ` section of the README, heading included, up to the next one."""
    start = readme.index(f"\n## {title}\n") + 1
    end = readme.find("\n## ", start + 1)
    return readme[start:] if end == -1 else readme[start : end + 1]


def notice() -> str:
    metadata = distribution("wordfreq").read_text("METADATA") or ""
    readme = metadata.split("\n\n", 1)[1]
    return HEAD + section(readme, "License") + "\n" + section(readme, "Citations to work that wordfreq is built on")


def main() -> None:
    here = Path(__file__).parent
    here.joinpath("NOTICE.md").write_text(notice(), encoding="utf-8", newline="\n")
    common: list[str] = []
    mid: list[str] = []
    digits: list[str] = []
    for key, freq in get_frequency_dict("de", wordlist="best").items():
        if MULTI_DIGIT_RE.search(key):
            digits.append(f"{key}\t{freq!r}")
            continue
        zipf = zipf_frequency(key, "de")
        if zipf >= 3.0:
            common.append(key)
        elif zipf >= 2.5:
            mid.append(key)
    lines = [
        "# wordfreq 3.1.1, de, wordlist best; written by generate.py, do not edit",
        "[common]",
        *sorted(common),
        "[mid]",
        *sorted(mid),
        "[digits]",
        *sorted(digits),
    ]
    data = ("\n".join(lines) + "\n").encode("utf-8")
    here.joinpath("de.txt.gz").write_bytes(gzip.compress(data, compresslevel=9, mtime=0))


if __name__ == "__main__":
    main()
