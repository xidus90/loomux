# /// script
# requires-python = ">=3.12"
# ///
"""SPIKE: MSYS grep with the artefact directories excluded, Windows."""

import statistics
import subprocess
import sys
import time

W = sys.argv[1]
G = r"C:\Program Files\Git\usr\bin\grep.exe"
EX = ["--exclude-dir=.git", "--exclude-dir=.loomux", "--exclude-dir=bin", "--exclude-dir=.cache"]
for name, pat, i in [("literal", "func Search", []), ("alternation", "Search|Grep", []),
                     ("ignore-case", "func search", ["-i"]), ("no-literal", r"[A-Z]{3}\w+", [])]:
    cmd = [G, "-rnE", *EX, *i, pat, "."]
    subprocess.run(cmd, cwd=W, capture_output=True)
    ts, out = [], b""
    for _ in range(7):
        t = time.perf_counter()
        out = subprocess.run(cmd, cwd=W, capture_output=True).stdout
        ts.append((time.perf_counter() - t) * 1000)
    print(f"{name:12} median {statistics.median(ts):7.1f} ms  lines {out.count(b'\n')}")
