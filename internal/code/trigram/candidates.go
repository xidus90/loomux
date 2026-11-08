package trigram

// Candidates narrows the corpus to the files that can contain one of the
// pattern's alternatives.
//
// Every alternative is a conjunction: all its literals have to appear, so the
// alternative's candidates are the intersection of their trigrams' postings.
// The alternatives form a disjunction, so their candidate sets are merged.
// A result of nil with used == false means "no condition, search everything";
// a nil or empty result with used == true means "no file can match". A broken
// posting list is "no condition" too, with every file.
func (ix *Index) Candidates(sets [][]string) (files []string, used bool) {
	if len(sets) == 0 {
		return ix.Files(), false
	}
	var union []uint32
	for _, set := range sets {
		if len(set) == 0 {
			// One alternative without a literal makes the disjunction
			// unconstrained.
			return ix.Files(), false
		}
		var inter []uint32
		first := true
		for _, literal := range set {
			grams := Grams([]byte(literal))
			if len(grams) == 0 {
				return ix.Files(), false
			}
			for _, gram := range grams {
				ids, ok := ix.Postings(gram)
				if !ok {
					// A broken posting list cannot narrow anything: a
					// short candidate set would drop hits silently.
					return ix.Files(), false
				}
				if first {
					inter, first = ids, false
					continue
				}
				inter = intersectIDs(inter, ids)
			}
			if len(inter) == 0 {
				break
			}
		}
		union = mergeIDs(union, inter)
	}
	files = make([]string, 0, len(union))
	for _, id := range union {
		files = append(files, ix.files[id])
	}
	return files, true
}

// intersectIDs intersects two ascending id lists.
func intersectIDs(left, right []uint32) []uint32 {
	out := make([]uint32, 0, len(left))
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		switch {
		case left[i] < right[j]:
			i++
		case left[i] > right[j]:
			j++
		default:
			out = append(out, left[i])
			i++
			j++
		}
	}
	return out
}

// mergeIDs unions two ascending id lists, keeping the order and dropping
// duplicates.
func mergeIDs(left, right []uint32) []uint32 {
	out := make([]uint32, 0, len(left)+len(right))
	i, j := 0, 0
	for i < len(left) || j < len(right) {
		var next uint32
		switch {
		case j >= len(right) || (i < len(left) && left[i] < right[j]):
			next = left[i]
			i++
		case i >= len(left) || right[j] < left[i]:
			next = right[j]
			j++
		default:
			next = left[i]
			i++
			j++
		}
		if len(out) == 0 || out[len(out)-1] != next {
			out = append(out, next)
		}
	}
	return out
}
