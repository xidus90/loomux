# /// script
# requires-python = ">=3.12"
# ///
"""SPIKE checks: full JSON equality base vs all switches, start floor timing,
tgrep stderr on a dirty tree."""

import json
import os
import statistics
import subprocess
import sys
import time
from pathlib import Path

W = Path(sys.argv[1]).resolve()
TGREP = r"C:\Users\micro\AppData\Local\Microsoft\WinGet\Packages\Microsoft.tgrep_Microsoft.Winget.Source_8wekyb3d8bbwe\tgrep.exe"
ALL = {f"LMX_SPIKE_{k}": "1" for k in ("PARWALK", "ONEWALK", "SPANS", "OVERLAY", "SCANPAR", "LAZYIDX", "RECGOB", "LAZYSPANS", "GCOFF", "LITPRE", "WHOLE")}
PATTERNS = [("func Search", False), ("Search|Grep", False), ("func search", True), (r"[A-Z]{3}\w+", False)]


def grep(exe, pattern, icase, extra_env):
    env = {k: v for k, v in os.environ.items() if not k.startswith("LMX_SPIKE_")} | extra_env
    cmd = [str(W / "bin" / exe), "graph", "grep", "--", pattern, "--json", "--channel", "cloud", "--max-hits", "20000"]
    if icase:
        cmd.append("-i")
    return json.loads(subprocess.run(cmd, cwd=W, env=env, capture_output=True, check=True).stdout)


def shape(r):
    groups = [(g["path"], (g.get("symbol") or {}).get("id"), g["in_degree"], tuple(h["line"] for h in g["hits"])) for g in r["groups"]]
    return groups


subprocess.run([str(W / "bin" / "spike.exe"), "graph", "build"], cwd=W, capture_output=True, check=True)
for pattern, icase in PATTERNS:
    a, b = grep("base.exe", pattern, icase, {}), grep("spike.exe", pattern, icase, ALL)
    sa, sb = shape(a), shape(b)
    pairs_a = {(p, l) for p, _, _, ls in sa for l in ls}
    pairs_b = {(p, l) for p, _, _, ls in sb for l in ls}
    print(f"{pattern!r:18} groups {len(sa)}/{len(sb)} identical={sa == sb} pairs {len(pairs_a)}/{len(pairs_b)} equal={pairs_a == pairs_b} index {a['index']}/{b['index']}")

for exe in ("base.exe", "spike.exe"):
    ts = []
    for _ in range(12):
        t = time.perf_counter()
        subprocess.run([str(W / "bin" / exe), "version"], capture_output=True)
        ts.append((time.perf_counter() - t) * 1000)
    print(f"version {exe}: median {statistics.median(ts[2:]):.1f} ms min {min(ts[2:]):.1f}")
ts = []
for _ in range(12):
    t = time.perf_counter()
    subprocess.run(["git", "--version"], capture_output=True)
    ts.append((time.perf_counter() - t) * 1000)
print(f"git --version (spawn floor): median {statistics.median(ts[2:]):.1f} ms")

f = W / "internal/brain/answer/answer.go"
saved = f.read_bytes()
try:
    with open(f, "a", encoding="utf-8") as h:
        h.write("\n// spike drift: func Search marker\n")
    p = subprocess.run([TGREP, "search", "--no-unicode", "--index-path", str(W / ".cache/tgrep-index"), "spike drift: func Search", "."], cwd=W, capture_output=True)
    print("tgrep dirty: exit", p.returncode, "stdout lines", p.stdout.count(b"\n"), "stderr", p.stderr[:400])
finally:
    f.write_bytes(saved)
