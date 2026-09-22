# Golden:  internal/brain/maintenance/testdata/stats.golden.tsv
# Call:    uv run --script stats_oracle.py <ultra-brain>/src <golden file>
#          The file is written in binary.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2); argv[1] is the `src` directory of a checkout at that tag.
# Source:  appendix of the task 10 report (stage 3a), unchanged.

# /// script
# requires-python = ">=3.12"
# dependencies = ["pyyaml"]
# ///
import sys, tempfile
from pathlib import Path
sys.path.insert(0, sys.argv[1])
from brain.maintenance.reconcile import _write_stats

STATS = {
    "Ärger.md": (1758300000000000000, 12),
    "a b.md": (0, 0),
    "a.md": (1, 7),
    "B.md": (4294967296, 4294967296),
    "z/tief/ä.md": (1758300000123456789, 1),
    "z/tief/zz.md": (2, 3),
}
with tempfile.TemporaryDirectory() as tmp:
    out = Path(tmp) / "maintenance" / "project-one" / "stats.tsv"
    _write_stats(out, STATS)
    Path(sys.argv[2]).write_bytes(out.read_bytes())
