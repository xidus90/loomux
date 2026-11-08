# /// script
# requires-python = ">=3.12"
# ///
"""Acceptance benchmark for the graph grep start-up work on Windows: an old
against a new loomux binary and tgrep, on two disposable clones of <repo>.

Usage: uv run --script acceptance.py <repo> <old.exe> <new.exe> <tgrep.exe> <out.json>

Scenarios: clean tree; 20 files changed after the build; the change committed;
--trust-index on the clean tree. One warm-up run, then 11 interleaved runs per
variant; the median counts. <repo> itself is never modified.
"""

import json
import os
import shutil
import stat
import statistics
import subprocess
import sys
import tempfile
import time
from pathlib import Path

REPO, OLD, NEW, TGREP = (Path(a).resolve() for a in sys.argv[1:5])
OUT = Path(sys.argv[5]).resolve()
RUNS = 11
DIRT_FILES = 20
MARKER = "\n// acceptance drift: func Search marker\n"
PATTERNS = {
    "literal": ("func Search", False),
    "alternation": ("Search|Grep", False),
    "ignore-case": ("func search", True),
    "no-literal": (r"[A-Z]{3}\w+", False),
}
# Medians in ms, 20 % tolerance included.
CHECK_LIMITS = {"literal": 108, "alternation": 132, "ignore-case": 114, "no-literal": 324}
DIRTY_LIMIT = 420
COMMIT_LIMIT = {"literal": 108}
TRUST_FACTOR = 1.2
TRUST_PATTERNS = ("literal", "alternation", "ignore-case")


def run(cmd, cwd, check=False):
    t = time.perf_counter()
    p = subprocess.run(cmd, cwd=cwd, capture_output=True, check=check)
    return (time.perf_counter() - t) * 1000, p


def git(tree, *args):
    return subprocess.run(["git", "-C", str(tree), *args], capture_output=True, text=True, check=True).stdout


def lmx(exe, pattern, icase, trust=False, max_hits=300):
    cmd = [str(exe), "graph", "grep", "--", pattern, "--json", "--channel", "cloud", "--max-hits", str(max_hits)]
    return cmd + (["-i"] if icase else []) + (["--trust-index"] if trust else [])


def tgrep(index, pattern, icase):
    return [str(TGREP), "search", "--no-unicode", "--index-path", str(index), *(["-i"] if icase else []), pattern, "."]


class Tree:
    """A clone of REPO with its own loomux state and a snapshot of it."""

    def __init__(self, work, name, exe):
        self.path, self.exe = work / name, exe
        self.state = self.path / ".loomux" / "state"
        self.snap = work / f"{name}-state"
        sha = git(REPO, "rev-parse", "HEAD").strip()
        subprocess.run(["git", "clone", "-q", "--shared", "--no-checkout", str(REPO), str(self.path)], check=True)
        git(self.path, "checkout", "-q", "--detach", sha)

    def build(self):
        ms, _ = run([str(self.exe), "graph", "build", "--no-reuse"], self.path, check=True)
        shutil.copytree(self.state, self.snap)
        return ms

    def restore(self):
        """Put the snapshot back, copying only the files a call changed, and
        read each copied file once. A full copy before every call left 85 MB
        of freshly written files whose first open (virus scan, file cache)
        landed in the measured call; a state a build wrote long ago has paid
        that (runs 1-6 used the full copy)."""
        want = set()
        for dirpath, _, names in os.walk(self.snap):
            rel = Path(dirpath).relative_to(self.snap)
            (self.state / rel).mkdir(parents=True, exist_ok=True)
            for n in names:
                src, dst = Path(dirpath) / n, self.state / rel / n
                want.add(dst)
                s = src.stat()
                try:
                    d = dst.stat()
                    if d.st_size == s.st_size and d.st_mtime_ns == s.st_mtime_ns:
                        continue
                except FileNotFoundError:
                    pass
                shutil.copy2(src, dst)
                dst.read_bytes()
        for dirpath, _, names in os.walk(self.state):
            for n in names:
                if (p := Path(dirpath) / n) not in want:
                    p.unlink()


def hits(p):
    try:
        r = json.loads(p.stdout)
        return f'{r["total_hits"] + r["truncated_hits"]} ({r.get("index")})'
    except ValueError as e:
        return f"error {e}: {p.stderr[:200]!r}"


def lines(p):
    return str(p.stdout.count(b"\n"))


def series(variants):
    """variants: name -> (cmd, cwd, hit counter, restore or None). Interleaved."""
    times = {k: [] for k in variants}
    last = {}
    for k, (cmd, cwd, _, restore) in variants.items():
        if restore:
            restore()
        run(cmd, cwd)
    for _ in range(RUNS):
        for k, (cmd, cwd, _, restore) in variants.items():
            if restore:
                restore()
            ms, last[k] = run(cmd, cwd)
            times[k].append(ms)
    return {k: {"median": round(statistics.median(times[k]), 1), "min": round(min(times[k]), 1),
                "max": round(max(times[k]), 1), "hits": variants[k][2](last[k])} for k in variants}


def full(tree, pattern, icase, trust=False):
    _, p = run(lmx(tree.exe, pattern, icase, trust, max_hits=20000), tree.path, check=True)
    return json.loads(p.stdout)


def shape(r):
    return [(g["path"], (g.get("symbol") or {}).get("id"), g["in_degree"], [h["line"] for h in g["hits"]])
            for g in r["groups"]]


def pairs(r):
    return sorted((g["path"], h["line"]) for g in r["groups"] for h in g["hits"])


def checks_clean(old, new):
    out = {}
    for name, (pattern, icase) in PATTERNS.items():
        a, b, t = full(old, pattern, icase), full(new, pattern, icase), full(new, pattern, icase, trust=True)
        strip = {"index", "index_built"}
        out[name] = {
            "old_eq_new_groups": shape(a) == shape(b),
            "new_eq_trusted": {k: v for k, v in b.items() if k not in strip} == {k: v for k, v in t.items() if k not in strip},
            "index": [a.get("index"), b.get("index"), t.get("index")],
        }
    return out


def checks_pairs(old, new):
    out = {}
    for name, (pattern, icase) in PATTERNS.items():
        old.restore()
        new.restore()
        out[name] = pairs(full(old, pattern, icase)) == pairs(full(new, pattern, icase))
    old.restore()
    new.restore()
    return out


def dirty(trees):
    files = [f for f in git(trees[0].path, "ls-files", "internal/brain/*.go").split() if not f.endswith("_test.go")]
    for tree in trees:
        for f in files[:DIRT_FILES]:
            with open(tree.path / f, "a", encoding="utf-8", newline="\n") as h:
                h.write(MARKER)
    return files[:DIRT_FILES]


def verdict(res):
    fails = []

    def need(label, value, limit):
        if value > limit:
            fails.append(f"{label}: {value} ms > {limit} ms")

    for p, lim in CHECK_LIMITS.items():
        need(f"clean {p} (check mode)", res["clean"][p]["new"]["median"], lim)
        need(f"after change {p}", res["dirty"][p]["new"]["median"], DIRTY_LIMIT)
    for p, lim in COMMIT_LIMIT.items():
        need(f"after commit {p} (first call)", res["commit"][p]["new first call"]["median"], lim)
    for p in TRUST_PATTERNS:
        row = res["clean"][p]
        need(f"clean {p} --trust-index", row["new --trust-index"]["median"], round(TRUST_FACTOR * row["tgrep"]["median"], 1))
    for p in PATTERNS:
        if not res["clean"][p]["new --trust-index"]["hits"].endswith("(trusted)"):
            fails.append(f"clean {p} --trust-index fell back: {res['clean'][p]['new --trust-index']['hits']}")
        for sc, variant in (("dirty", "new"), ("commit", "new first call"), ("commit", "new later calls")):
            if not res[sc][p][variant]["hits"].endswith("(overlay)"):
                fails.append(f"{sc} {p} {variant} did not answer from the overlay: {res[sc][p][variant]['hits']}")
    for sc, checks in res["checks"].items():
        for p, ok in checks.items():
            good = ok if isinstance(ok, bool) else ok["old_eq_new_groups"] and ok["new_eq_trusted"]
            if not good:
                fails.append(f"hits differ: {sc} {p} {ok}")
    return fails


def main():
    work = Path(tempfile.mkdtemp(prefix="lmx-acceptance-"))
    try:
        old, new = Tree(work, "old", OLD), Tree(work, "new", NEW)
        tidx = work / "tgrep-index"
        res = {"repo_head": git(REPO, "rev-parse", "HEAD").strip(), "runs": RUNS,
               "files": int(git(new.path, "ls-files").count("\n")),
               "build": {"old graph build": old.build(), "new graph build": new.build()},
               "checks": {}, "clean": {}, "dirty": {}, "commit": {}}
        res["build"]["tgrep index"] = run([str(TGREP), "index", "--index-path", str(tidx), "."], new.path, check=True)[0]
        res["checks"]["clean"] = checks_clean(old, new)

        for name, (pattern, icase) in PATTERNS.items():
            res["clean"][name] = series({
                "old": (lmx(OLD, pattern, icase), old.path, hits, None),
                "new": (lmx(NEW, pattern, icase), new.path, hits, None),
                "new --trust-index": (lmx(NEW, pattern, icase, trust=True), new.path, hits, None),
                "tgrep": (tgrep(tidx, pattern, icase), new.path, lines, None),
            })
            print("clean", name, {k: v["median"] for k, v in res["clean"][name].items()}, flush=True)

        res["dirty_files"] = dirty((old, new))
        res["checks"]["dirty"] = checks_pairs(old, new)
        for name, (pattern, icase) in PATTERNS.items():
            # Every call is the first grep after the edit: the state is the clean build.
            res["dirty"][name] = series({
                "old": (lmx(OLD, pattern, icase), old.path, hits, old.restore),
                "new": (lmx(NEW, pattern, icase), new.path, hits, new.restore),
            })
            print("dirty", name, {k: v["median"] for k, v in res["dirty"][name].items()}, flush=True)

        for tree in (old, new):
            git(tree.path, "add", "-A")
            git(tree.path, "-c", "user.name=bench", "-c", "user.email=bench@localhost", "commit", "-qm", "drift")
        res["checks"]["commit"] = checks_pairs(old, new)
        for name, (pattern, icase) in PATTERNS.items():
            res["commit"][name] = series({
                "old": (lmx(OLD, pattern, icase), old.path, hits, old.restore),
                "new first call": (lmx(NEW, pattern, icase), new.path, hits, new.restore),
            })
            # Steady state: the first call stored the new git index stamp.
            new.restore()
            res["commit"][name] |= series({"new later calls": (lmx(NEW, pattern, icase), new.path, hits, None)})
            print("commit", name, {k: v["median"] for k, v in res["commit"][name].items()}, flush=True)

        res["failed"] = verdict(res)
        OUT.write_text(json.dumps(res, indent=2), encoding="utf-8")
        print("checks", json.dumps(res["checks"]))
        print("FAILED:" if res["failed"] else "PASSED", *res["failed"], sep="\n  ")
    finally:
        shutil.rmtree(work, onexc=lambda f, p, _: (os.chmod(p, stat.S_IWRITE), f(p)))


main()
