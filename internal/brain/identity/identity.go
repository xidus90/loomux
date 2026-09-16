package identity

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// CrockfordAlphabet is the 32-character Crockford Base32 alphabet used by ULIDs.
const CrockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// IdentitiesHeader is the tab-separated TSV column header for _identities.tsv.
const IdentitiesHeader = "doc_id\tpfad\tcontent_hash\trevision"

// Identity represents a stable tracking entry for a document across edits and renames.
type Identity struct {
	DocID       string
	Relative    string
	ContentHash string
	Revision    int
}

func encodeBase32(val uint64, length int) string {
	buf := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		buf[i] = CrockfordAlphabet[val%32]
		val /= 32
	}
	return string(buf)
}

func encodeBigIntBase32(val *big.Int, length int) string {
	radix := big.NewInt(32)
	rem := new(big.Int)
	v := new(big.Int).Set(val)
	buf := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		v.DivMod(v, radix, rem)
		buf[i] = CrockfordAlphabet[rem.Int64()]
	}
	return string(buf)
}

// NewDocID generates a 26-character time-ordered Crockford Base32 ULID.
// 48 bits of millisecond timestamp ensure chronological register ordering;
// 80 bits of cryptographic randomness avoid counter coordination across processes.
func NewDocID() string {
	ms := uint64(time.Now().UnixMilli())
	timePart := encodeBase32(ms, 10)

	var randBytes [10]byte
	_, _ = rand.Read(randBytes[:])
	randVal := new(big.Int).SetBytes(randBytes[:])
	randPart := encodeBigIntBase32(randVal, 16)

	return timePart + randPart
}

// ContentHash computes the SHA-256 digest of file bytes with CRLF normalized to LF.
// Normalizing line endings ensures checkouts on different platforms (Windows vs Linux)
// do not invalidate content hashes or trigger spurious modification reports (Spec 5.6).
func ContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	sum := sha256.Sum256(normalized)
	return fmt.Sprintf("sha256:%x", sum), nil
}

// ReadIdentities loads the identity TSV file into memory, the way
// identity.read_identities does (src/brain/identity.py:55-77).
// Returns an empty map if the file does not exist; like Path.exists(), any
// failure to stat the path counts as "does not exist".
// Returns an error if the file is not UTF-8 or a row is malformed or has an
// invalid revision.
func ReadIdentities(path string) (map[string]Identity, error) {
	if _, err := os.Stat(path); err != nil {
		return map[string]Identity{}, nil
	}
	text, err := pytext.ReadText(path)
	if err != nil {
		return nil, err
	}

	lines := pytext.SplitLines(text)
	identities := make(map[string]Identity)

	// Line numbers start at 2 to count the header line and match editor line displays
	for i, line := range lines {
		if i == 0 {
			continue
		}
		if pytext.Strip(line) == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		lineNum := i + 1
		if len(fields) != 4 {
			return nil, fmt.Errorf("%s: line %d: expected 4 tab-separated fields, found %d", path, lineNum, len(fields))
		}

		docID := fields[0]
		relative := fields[1]
		digest := fields[2]
		revisionStr := fields[3]

		rev, ok := parseRevision(revisionStr)
		if !ok {
			return nil, fmt.Errorf("%s: line %d: revision %s is not a number", path, lineNum, pytext.Repr(revisionStr))
		}

		identities[relative] = Identity{
			DocID:       docID,
			Relative:    relative,
			ContentHash: digest,
			Revision:    rev,
		}
	}

	return identities, nil
}

// parseRevision accepts what Python's revision.isdigit() and int() accept,
// narrowed to ASCII digits and to the range of int: a sign, a space or an
// empty field is refused as Python refuses it; a non-ASCII digit and a value
// beyond int are refused where Python would read or crash (parity list).
func parseRevision(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	rev, err := strconv.Atoi(s)
	return rev, err == nil
}

// MatchRenames identifies files whose paths vanished while identical content hashes
// appeared at new paths (Spec 5.6).
// To prevent incorrect identity assignment in the presence of duplicate contents,
// renames are only matched when exactly one vanished file and one fresh file share that hash.
func MatchRenames(previous map[string]Identity, current map[string]string) map[string]Identity {
	gone := make(map[string]Identity)
	for path, id := range previous {
		if _, ok := current[path]; !ok {
			gone[path] = id
		}
	}

	fresh := make(map[string]string)
	for path, hash := range current {
		if _, ok := previous[path]; !ok {
			fresh[path] = hash
		}
	}

	byHash := make(map[string][]Identity)
	for _, id := range gone {
		byHash[id.ContentHash] = append(byHash[id.ContentHash], id)
	}

	arrivalsByHash := make(map[string]int)
	for _, hash := range fresh {
		arrivalsByHash[hash]++
	}

	matched := make(map[string]Identity)
	for path, hash := range fresh {
		candidates := byHash[hash]
		if len(candidates) == 1 && arrivalsByHash[hash] == 1 {
			matched[path] = Identity{
				DocID:       candidates[0].DocID,
				Relative:    path,
				ContentHash: hash,
				Revision:    candidates[0].Revision,
			}
		}
	}

	return matched
}

// RenderIdentities serializes identities into TSV format sorted by doc_id
// for deterministic diffing without churn.
func RenderIdentities(identities map[string]Identity) string {
	rows := make([]Identity, 0, len(identities))
	for _, id := range identities {
		rows = append(rows, id)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].DocID < rows[j].DocID
	})

	var sb strings.Builder
	sb.WriteString(IdentitiesHeader)
	sb.WriteString("\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\n", r.DocID, r.Relative, r.ContentHash, r.Revision))
	}
	return sb.String()
}
