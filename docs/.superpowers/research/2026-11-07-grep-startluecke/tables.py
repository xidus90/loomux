# /// script
# requires-python = ">=3.12"
# ///
"""Render the spike benchmark JSON as markdown tables (median, hits)."""

import json
import sys

r = json.load(open(sys.argv[1], encoding="utf-8"))
COLS = ["base", "S1 parwalk", "S1+S2 onewalk", "S1+S2+S3 spans", "S1-S4 overlay", "S1-S5 scanpar",
        "tgrep", "rg", "git grep", "findstr /s", "grep -rn (MSYS)"]
HEAD = ["Basis", "S1", "S1+S2", "S1–S3", "S1–S4", "S1–S5", "tgrep", "rg", "git grep", "findstr /s", "grep -rn (MSYS)"]
NAMES = {"literal": "Literal `func Search`", "alternation": "Alternation `Search|Grep`",
         "ignore-case": "`-i func search`", "no-literal": "ohne Literal `[A-Z]{3}\\w+`"}
for sc in ("clean", "dirty"):
    print(f"\n#### {sc}\n")
    print("| Muster | " + " | ".join(HEAD) + " |")
    print("|---|" + "---:|" * len(HEAD))
    for p, row in r[sc].items():
        cells = [f"{row[c]['median']:.0f}" for c in COLS]
        print(f"| {NAMES[p]} | " + " | ".join(cells) + " |")
    print("\nTreffer (`total_hits + truncated_hits` bei loomux, Ausgabezeilen sonst):\n")
    print("| Muster | loomux (alle Varianten) | tgrep | rg | git grep | findstr /s | grep -rn (MSYS) |")
    print("|---|---:|---:|---:|---:|---:|---:|")
    for p, row in r[sc].items():
        lx = {row[c]["hits"] for c in COLS[:6]}
        lx = " / ".join(sorted(lx))
        print(f"| {NAMES[p]} | {lx} | " + " | ".join(row[c]["hits"] for c in COLS[6:]) + " |")
print("\nbuild", {k: round(v) for k, v in r["build"].items()})
