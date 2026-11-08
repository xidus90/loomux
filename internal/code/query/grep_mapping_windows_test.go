package query

import (
	"os"
	"sync"
	"testing"
)

// The scan reads every candidate file and has no bound; a build that has to
// replace the index or remove the symbol spans waits about 2 s on Windows. So
// neither may be mapped while the scan runs. Windows refuses to truncate a
// mapped file, which is what this probes from inside the scan.
func TestGrepMapsNothingDuringTheScan(t *testing.T) {
	for name, trust := range map[string]bool{"checked": false, "trusted": true} {
		t.Run(name, func(t *testing.T) {
			root := grepFixture(t)
			var once sync.Once
			var mapped []error
			keep := func(string) bool {
				once.Do(func() {
					for _, p := range []string{indexPath(root), spansPath(root)} {
						if err := os.Truncate(p, 0); err != nil {
							mapped = append(mapped, err)
						}
					}
				})
				return true
			}
			res, notes, err := Grep(root, "func Run", GrepOptions{Keep: keep, TrustIndex: trust})
			if err != nil {
				t.Fatal(err)
			}
			if len(mapped) > 0 || res.TotalHits == 0 {
				t.Fatalf("still mapped during the scan: %v (hits %d, index %q, notes %v)", mapped, res.TotalHits, res.Index, notes)
			}
		})
	}
}
