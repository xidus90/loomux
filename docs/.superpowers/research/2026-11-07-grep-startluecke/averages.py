# /// script
# requires-python = ">=3.12"
# ///
"""Average the acceptance runs: loomux time as a percentage of tgrep (clean
tree) and of the old binary (all scenarios), over all runs and split by load.
Usage: uv run --script averages.py
High load: the runs the benchmark page records as loaded (CPU at 80-100 %
or graph builds well above 20 s); all other runs count as low load.
"""
import json
import statistics
from pathlib import Path

HERE = Path(__file__).parent
RUNS = range(1, 16)
HIGH = {2, 4, 5, 6, 7, 10}
PATTERNS = ("literal", "alternation", "ignore-case", "no-literal")


def load(n):
    return json.loads((HERE / f"acceptance-windows-run{n}.json").read_text(encoding="utf-8"))


def median(d, scenario, pattern, variant):
    return d[scenario][pattern][variant]["median"]


def ratios(runs):
    """Per pattern: mean of the per-run percentages."""
    rows = {}
    for p in PATTERNS:
        cols = {"check/tgrep": [], "trust/tgrep": [], "check/old": [], "dirty/old": [], "commit/old": []}
        for n in runs:
            d = load(n)
            t = median(d, "clean", p, "tgrep")
            cols["check/tgrep"].append(100 * median(d, "clean", p, "new") / t)
            cols["trust/tgrep"].append(100 * median(d, "clean", p, "new --trust-index") / t)
            cols["check/old"].append(100 * median(d, "clean", p, "new") / median(d, "clean", p, "old"))
            cols["dirty/old"].append(100 * median(d, "dirty", p, "new") / median(d, "dirty", p, "old"))
            cols["commit/old"].append(100 * median(d, "commit", p, "new first call") / median(d, "commit", p, "old"))
        rows[p] = {k: round(statistics.mean(v), 1) for k, v in cols.items()}
    return rows


if __name__ == "__main__":
    groups = {
        "all": list(RUNS),
        "low": [n for n in RUNS if n not in HIGH],
        "high": [n for n in RUNS if n in HIGH],
    }
    print(json.dumps({name: {"runs": runs, "percent": ratios(runs)} for name, runs in groups.items()}, indent=1))
