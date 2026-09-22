# Golden:  internal/brain/maintenance/testdata/hunks-fuzz.golden.json
# Call:    from the ultra-brain checkout:
#          uv run python hunks_fuzz_oracle.py <golden file>
#          The file is written in binary. Seed and distributions are fixed, so
#          every run writes the same corpus, inputs included.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), imported from the checkout named in sys.path below, which
#          has to stand at that tag.
# Source:  appendix of the task 11 report (stage 3a, fix round 1), unchanged.

# /// script
# requires-python = ">=3.12"
# ///
import json, random, sys
sys.path.insert(0, r"C:\Users\micro\Documents\#GIT\ultra-brain\src")
from brain.maintenance.reconcile import _Changed, _hunks

rnd = random.Random(20260920)
def gen(n, alpha):
    return "".join(rnd.choice(alpha) + "\n" for _ in range(n))

vectors = {}
for case in range(400):
    alpha = [f"L{i}" for i in range(rnd.choice([2, 3, 5, 12, 40]))]
    n = rnd.choice([0, 1, 3, 8, 30, 120, 210, 260])
    m = rnd.choice([0, 1, 3, 8, 30, 120, 210, 260])
    a, b = gen(n, alpha), gen(m, alpha)
    if rnd.random() < 0.2 and b:
        b = b[:-1]
    vectors[f"c{case}"] = (a, b)

out = {k: _hunks(_Changed(doc_id="d", relative="src/a.go", revision=1,
                          content_hash="sha256:00", text=b, baseline=a))
       for k, (a, b) in vectors.items()}
with open(sys.argv[1], "wb") as h:
    h.write(json.dumps({"inputs": vectors, "hunks": out},
                       ensure_ascii=False, sort_keys=True).encode("utf-8"))
