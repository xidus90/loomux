# Goldens: internal/brain/wiki/testdata/bundle/*.golden (five files)
# Call:    uv run --script bundle_oracle.py <ultra-brain>/src internal/brain/wiki/testdata/bundle
#          Every file is written in binary.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2); argv[1] is the `src` directory of a checkout at that tag,
#          and the script calls `init_bundle` of src/brain/wiki/scaffold.py.
# Source:  appendix of the task 15 report (stage 3a), unchanged.

# /// script
# requires-python = ">=3.12"
# ///
"""Write the bundle frame of `init_bundle` as goldens.

argv[1]: ultra-brain's `src` directory; argv[2]: the golden directory.
"""
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, sys.argv[1])
from brain.wiki.scaffold import init_bundle

out = Path(sys.argv[2])
out.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory() as tmp:
    for written in init_bundle(Path(tmp) / "wiki"):
        (out / (written.name + ".golden")).write_bytes(written.read_bytes())
        print(written.name)
