# /// script
# requires-python = ">=3.12"
# ///
"""loomux graph grep against tgrep and zvec-grep (zg) on Windows, on one
disposable clone of <repo>.
Usage: uv run --script zvec-vergleich.py <repo> <loomux.exe> <tgrep.exe> <zg.cmd> <out.json>
Variants: loomux in check mode and with --trust-index, tgrep, zg --rg (managed
ripgrep, no index), zg --fts in direct mode (one process per call) and in
server mode (warm daemon). zg --fts is ranked BM25, not a regex, so it only
runs the patterns that are plain words. One warm-up run, then 11 interleaved
runs per variant; the median counts. <repo> itself is never modified.
"""
import json
import os
import shutil
import statistics
import subprocess
import sys
import tempfile
import time
from pathlib import Path

REPO, LMX, TGREP, ZG = (Path(a).resolve() for a in sys.argv[1:5])
OUT = Path(sys.argv[5]).resolve()
RUNS = 11
PATTERNS = {
    "literal": ("func Search", False),
    "alternation": ("Search|Grep", False),
    "ignore-case": ("func search", True),
    "no-literal": (r"[A-Z]{3}\w+", False),
}
FTS_PATTERNS = ("literal", "ignore-case")


def run(cmd, cwd, env=None, check=False):
    t = time.perf_counter()
    p = subprocess.run(cmd, cwd=cwd, capture_output=True, env=env, check=check)
    return (time.perf_counter() - t) * 1000, p


def lines(p):
    return str(p.stdout.count(b"\n"))


def lmx_hits(p):
    try:
        r = json.loads(p.stdout)
        return f'{r["total_hits"] + r["truncated_hits"]} ({r.get("index")})'
    except ValueError as e:
        return f"error {e}: {p.stderr[:200]!r}"


def series(variants):
    times = {k: [] for k in variants}
    last = {}
    for cmd, cwd, env, _ in variants.values():
        run(cmd, cwd, env)
    for _ in range(RUNS):
        for k, (cmd, cwd, env, _) in variants.items():
            ms, last[k] = run(cmd, cwd, env)
            times[k].append(ms)
    return {k: {"median": round(statistics.median(times[k]), 1), "min": round(min(times[k]), 1),
                "max": round(max(times[k]), 1), "hits": variants[k][3](last[k])} for k in variants}


def main():
    work = Path(tempfile.mkdtemp(prefix="zvec-vergleich-"))
    tree = work / "tree"
    env = dict(os.environ, ZVEC_GREP_HOME=str(work / "zghome"), NO_COLOR="1")
    zg = [str(ZG)]
    try:
        subprocess.run(["git", "clone", "-q", "--shared", str(REPO), str(tree)], check=True)
        res = {"repo_head": subprocess.run(["git", "-C", str(tree), "rev-parse", "HEAD"], capture_output=True,
                                           text=True, check=True).stdout.strip(), "runs": RUNS, "build": {}}
        tidx = work / "tgrep-index"
        res["build"]["loomux graph build"] = run([str(LMX), "graph", "build", "--no-reuse"], tree, check=True)[0]
        res["build"]["tgrep index"] = run([str(TGREP), "index", "--index-path", str(tidx), "."], tree, check=True)[0]
        res["build"]["zg index"] = run(zg + ["--index", "--mode", "direct"], tree, env, check=True)[0]
        run(zg + ["--server", "on"], tree, env, check=True)
        res["clean"] = {}
        for name, (pattern, icase) in PATTERNS.items():
            i = ["-i"] if icase else []
            lmx = [str(LMX), "graph", "grep", "--", pattern, "--json", "--channel", "cloud", "--max-hits", "300", *i]
            variants = {
                "loomux": (lmx, tree, None, lmx_hits),
                "loomux --trust-index": (lmx + ["--trust-index"], tree, None, lmx_hits),
                "tgrep": ([str(TGREP), "search", "--no-unicode", "--index-path", str(tidx), *i, pattern, "."],
                          tree, None, lines),
                "zg --rg": (zg + ["--rg", "-n", *i, pattern, "."], tree, env, lines),
            }
            if name in FTS_PATTERNS:
                fts = zg + ["--fts", pattern, "--preview", "none", "--limit", "300", "--color", "never"]
                variants["zg --fts direct"] = (fts + ["--mode", "direct"], tree, env, lines)
                variants["zg --fts server"] = (fts + ["--mode", "server"], tree, env, lines)
            res["clean"][name] = series(variants)
            print(name, {k: v["median"] for k, v in res["clean"][name].items()}, flush=True)
        OUT.write_text(json.dumps(res, indent=1), encoding="utf-8")
    finally:
        subprocess.run(zg + ["--server", "off"], cwd=tree, env=env, capture_output=True)
        shutil.rmtree(work, ignore_errors=True)


if __name__ == "__main__":
    main()
