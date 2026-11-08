# /// script
# requires-python = ">=3.12"
# ///
"""SPIKE: loomux S1-S12 with and without --no-refresh + TRUST against tgrep,
interleaved runs, median of 11."""

import json
import os
import statistics
import subprocess
import sys
import time
from pathlib import Path

W = Path(sys.argv[1]).resolve()
TGREP = r"C:\Users\micro\AppData\Local\Microsoft\WinGet\Packages\Microsoft.tgrep_Microsoft.Winget.Source_8wekyb3d8bbwe\tgrep.exe"
TIDX = W / ".cache" / "tgrep-index"
PATTERNS = {"literal": ("func Search", False), "alternation": ("Search|Grep", False),
            "ignore-case": ("func search", True), "no-literal": (r"[A-Z]{3}\w+", False)}
ALL = ("PARWALK", "ONEWALK", "SPANS", "OVERLAY", "SCANPAR", "LAZYIDX", "RECGOB", "LAZYSPANS", "GCOFF", "LITPRE", "WHOLE")


def env(extra):
    e = {k: v for k, v in os.environ.items() if not k.startswith("LMX_SPIKE_")}
    e.update({f"LMX_SPIKE_{k}": "1" for k in ALL + extra})
    return e


def lmx(p, i, trust):
    c = [str(W / "bin" / "spike.exe"), "graph", "grep", "--", p, "--json", "--channel", "cloud"]
    c += ["-i"] if i else []
    c += ["--max-hits", "300"]
    if trust:
        c.append("--no-refresh")
    return c


def run(cmd, e=None):
    t = time.perf_counter()
    p = subprocess.run(cmd, cwd=W, env=e, capture_output=True)
    return (time.perf_counter() - t) * 1000, p


out = {}
for name, (p, i) in PATTERNS.items():
    variants = {
        "S1-S12 (Pruefung an)": (lmx(p, i, False), env(())),
        "an + MMAP/SPANIDX": (lmx(p, i, False), env(("MMAP", "SPANIDX"))),
        "aus: TRUSTIX+MMAP+SPANIDX": (lmx(p, i, True), env(("TRUST", "TRUSTIX", "MMAP", "SPANIDX"))),
        "tgrep": ([TGREP, "search", "--no-unicode", "--index-path", str(TIDX), *(["-i"] if i else []), p, "."], None),
    }
    times = {k: [] for k in variants}
    last = {}
    for k, (c, e) in variants.items():
        run(c, e)
    for _ in range(11):
        for k, (c, e) in variants.items():
            ms, last[k] = run(c, e)
            times[k].append(ms)
    row = {}
    for k in variants:
        o = last[k].stdout
        if k == "tgrep":
            hits = o.count(b"\n")
        else:
            try:
                r = json.loads(o)
                hits = f'{r["total_hits"] + r["truncated_hits"]} ({r.get("index")})'
            except Exception as ex:  # noqa: BLE001 - spike
                hits = f"error {ex} {last[k].stderr[:200]!r}"
        row[k] = {"median": round(statistics.median(times[k]), 1), "min": round(min(times[k]), 1), "hits": hits}
    out[name] = row
    print(name, json.dumps(row), flush=True)
