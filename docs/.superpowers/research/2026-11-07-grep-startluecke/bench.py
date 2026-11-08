# /// script
# requires-python = ">=3.12"
# ///
"""SPIKE benchmark: loomux graph grep variants against tgrep, rg, git grep,
findstr and MSYS grep, on a clean and on a dirty tree.

Usage: uv run --script bench.py <worktree> <out.json>
"""

import json
import os
import shutil
import statistics
import subprocess
import sys
import time
from pathlib import Path

W = Path(sys.argv[1]).resolve()
OUT = Path(sys.argv[2])
STATE = W / ".loomux" / "state" / "graph"
SNAP = OUT.parent / "graph-snapshot"
TGREP = r"C:\Users\micro\AppData\Local\Microsoft\WinGet\Packages\Microsoft.tgrep_Microsoft.Winget.Source_8wekyb3d8bbwe\tgrep.exe"
TIDX = W / ".cache" / "tgrep-index"
MSYS_GREP = r"C:\Program Files\Git\usr\bin\grep.exe"
RUNS = 7
DIRTY_RUNS = 5

PATTERNS = {
    "literal": ("func Search", False),
    "alternation": ("Search|Grep", False),
    "ignore-case": ("func search", True),
    "no-literal": (r"[A-Z]{3}\w+", False),
}
FINDSTR = {
    "literal": ["/c:func Search"],
    "alternation": ["/r", "Search Grep"],
    "ignore-case": ["/i", "/c:func search"],
    "no-literal": ["/r", "[A-Z][A-Z][A-Z][a-zA-Z0-9_]"],
}
VARIANTS = {
    "base": ("base.exe", {}),
    "S1 parwalk": ("spike.exe", {"PARWALK": "1"}),
    "S1+S2 onewalk": ("spike.exe", {"PARWALK": "1", "ONEWALK": "1"}),
    "S1+S2+S3 spans": ("spike.exe", {"PARWALK": "1", "ONEWALK": "1", "SPANS": "1"}),
    "S1-S4 overlay": ("spike.exe", {"PARWALK": "1", "ONEWALK": "1", "SPANS": "1", "OVERLAY": "1"}),
    "S1-S5 scanpar": ("spike.exe", {"PARWALK": "1", "ONEWALK": "1", "SPANS": "1", "OVERLAY": "1", "SCANPAR": "1"}),
    "S1-S10": ("spike.exe", {k: "1" for k in ("PARWALK", "ONEWALK", "SPANS", "OVERLAY", "SCANPAR",
                                               "LAZYIDX", "RECGOB", "LAZYSPANS", "GCOFF")}),
    "S1-S12": ("spike.exe", {k: "1" for k in ("PARWALK", "ONEWALK", "SPANS", "OVERLAY", "SCANPAR",
                                               "LAZYIDX", "RECGOB", "LAZYSPANS", "GCOFF", "LITPRE", "WHOLE")}),
}
if os.environ.get("BENCH_ONLY"):
    VARIANTS = {k: v for k, v in VARIANTS.items() if k in os.environ["BENCH_ONLY"].split(",")}


def loomux_cmd(exe, pattern, icase):
    cmd = [str(W / "bin" / exe), "graph", "grep", "--", pattern, "--json", "--channel", "cloud"]
    if icase:
        cmd.append("-i")
    return cmd + ["--max-hits", "300"]


def env_for(flags):
    env = {k: v for k, v in os.environ.items() if not k.startswith("LMX_SPIKE_")}
    env.update({f"LMX_SPIKE_{k}": v for k, v in flags.items()})
    return env


def run(cmd, env=None):
    t = time.perf_counter()
    p = subprocess.run(cmd, cwd=W, env=env, capture_output=True)
    return (time.perf_counter() - t) * 1000, p


def loomux_hits(p):
    try:
        r = json.loads(p.stdout)
        return f'{r["total_hits"] + r["truncated_hits"]} ({r.get("index")})'
    except Exception as e:  # noqa: BLE001 - spike: report, not raise
        return f"error {e}: {p.stderr[:200]!r}"


def line_hits(p):
    return str(p.stdout.count(b"\n"))


def series(cmd, env, hits, n=RUNS):
    run(cmd, env)
    times, last = [], None
    for _ in range(n):
        ms, last = run(cmd, env)
        times.append(ms)
    return {"median": statistics.median(times), "min": min(times), "max": max(times), "hits": hits(last)}


def baselines(pname):
    pattern, icase = PATTERNS[pname]
    i = ["-i"] if icase else []
    return {
        "tgrep": (["tgrep_placeholder"], None),
        "rg": (["rg", "-n", *i, pattern], line_hits),
        "git grep": (["git", "grep", "-n", "-E", *i, pattern], line_hits),
        "findstr /s": (["findstr", "/s", "/n", *FINDSTR[pname], "*"], line_hits),
        "grep -rn (MSYS)": ([MSYS_GREP, "-rnE", *i, pattern, "."], line_hits),
    }


def tgrep_cmd(pattern, icase):
    return [TGREP, "search", "--no-unicode", "--index-path", str(TIDX), *(["-i"] if icase else []), pattern, "."]


DIRT = []


def dirty_files():
    # Go sources under internal/brain: graph sources, and never a file a spike edits.
    out = subprocess.run(["git", "ls-files", "internal/brain/*.go"], cwd=W, capture_output=True, text=True).stdout.split()
    return [f for f in out if not f.endswith("_test.go")][:20]


SAVED = {}


def apply_dirt():
    for f in DIRT:
        SAVED[f] = (W / f).read_bytes()
        with open(W / f, "a", encoding="utf-8") as h:
            h.write("\n// spike drift: func Search marker\n")


def revert_dirt():
    for f, data in SAVED.items():
        (W / f).write_bytes(data)


def snapshot():
    if SNAP.exists():
        shutil.rmtree(SNAP)
    shutil.copytree(STATE, SNAP)


def restore():
    shutil.rmtree(STATE)
    shutil.copytree(SNAP, STATE)


def main():
    global DIRT
    DIRT = dirty_files()
    results = {"clean": {}, "dirty": {}, "build": {}}
    ms, p = run([str(W / "bin" / "spike.exe"), "graph", "build", "--no-reuse"])
    results["build"]["loomux graph build --no-reuse"] = ms
    ms, p = run([TGREP, "index", "--index-path", str(TIDX), "."])
    results["build"]["tgrep index"] = ms
    snapshot()

    for scenario in os.environ.get("BENCH_SCENARIOS", "clean,dirty").split(","):
        if scenario == "dirty":
            apply_dirt()
        for pname, (pattern, icase) in PATTERNS.items():
            row = {}
            for vname, (exe, flags) in VARIANTS.items():
                cmd, env = loomux_cmd(exe, pattern, icase), env_for(flags)
                if scenario == "clean":
                    restore()
                    row[vname] = series(cmd, env, loomux_hits)
                else:
                    # First grep after an edit: every call starts from the
                    # snapshot built on the clean tree.
                    times, last = [], None
                    for _ in range(DIRTY_RUNS):
                        restore()
                        ms, last = run(cmd, env)
                        times.append(ms)
                    row[vname] = {"median": statistics.median(times), "min": min(times), "max": max(times), "hits": loomux_hits(last)}
            for bname, (cmd, hits) in baselines(pname).items():
                if bname == "tgrep":
                    cmd, hits = tgrep_cmd(pattern, icase), line_hits
                row[bname] = series(cmd, None, hits)
            results[scenario][pname] = row
            print(scenario, pname, {k: round(v["median"], 1) for k, v in row.items()}, flush=True)
        if scenario == "dirty":
            revert_dirt()
    restore()
    OUT.write_text(json.dumps(results, indent=2), encoding="utf-8")


main()
