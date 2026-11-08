# /// script
# requires-python = ">=3.12"
# ///
"""SPIKE: JSON of check-on vs check-off (TRUSTIX) must agree on the hits."""

import json
import os
import subprocess
import sys
from pathlib import Path

W = Path(sys.argv[1]).resolve()
ALL = ("PARWALK", "ONEWALK", "SPANS", "OVERLAY", "SCANPAR", "LAZYIDX", "RECGOB", "LAZYSPANS", "GCOFF", "LITPRE", "WHOLE", "MMAP", "SPANIDX")
PATTERNS = [("func Search", False), ("Search|Grep", False), ("func search", True), (r"[A-Z]{3}\w+", False)]


def run(p, i, off):
    e = {k: v for k, v in os.environ.items() if not k.startswith("LMX_SPIKE_")}
    e.update({f"LMX_SPIKE_{k}": "1" for k in ALL + (("TRUST", "TRUSTIX") if off else ())})
    c = [str(W / "bin" / "spike.exe"), "graph", "grep", "--", p, "--json", "--channel", "cloud", "--max-hits", "300"]
    c += ["-i"] if i else []
    c += ["--no-refresh"] if off else []
    return json.loads(subprocess.run(c, cwd=W, env=e, capture_output=True).stdout)


for p, i in PATTERNS:
    on, off = run(p, i, False), run(p, i, True)
    diff = sorted(k for k in set(on) | set(off) if on.get(k) != off.get(k))
    print(p, "-i" if i else "", "differing keys:", diff, {k: (on.get(k), off.get(k)) for k in diff if k != "hits"})
