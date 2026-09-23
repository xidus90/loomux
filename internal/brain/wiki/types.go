package wiki

// Rank is where a page type draws its validity from, in descending
// bindingness (`src/brain/wiki/types.py`). The strings are the ones the
// census prints.
type Rank string

const (
	RankCore      Rank = "core"
	RankCatalogue Rank = "catalogue"
	RankOrigin    Rank = "origin"
	RankDeclared  Rank = "declared"
	RankUnknown   Rank = "unknown"
)

// coreTypes, catalogueTypes and originTypes are CORE_TYPES, CATALOGUE_TYPES
// and ORIGIN_TYPES of types.py, spelled as it spells them: a type is asked
// letter for letter, so that `Architecure` stands out. The core is measured
// on two stocks, the catalogue is binding in name wherever its case occurs,
// and the origin types are what an ingest writes.
//
// Functions and not package variables: the sets are small literals, and a
// function keeps the start path free of anything built at load.
func coreTypes() map[string]bool {
	return map[string]bool{"Architecture": true, "Decision": true, "Open Question": true, "Reference": true}
}

func catalogueTypes() map[string]bool {
	return map[string]bool{"API Endpoint": true, "Data Model": true, "Metric": true, "Runbook": true, "Glossary Entry": true}
}

func originTypes() map[string]bool {
	return map[string]bool{"Source": true, "Topic": true, "Entity": true, "Synthesis": true}
}

// RankOf is `rank_of`: the rank of a page type in an area that declares
// the given types of its own. A missing type is RankUnknown and not a rank
// of its own, because a page without a type and a page with a made-up one
// are the same case for every rule that asks.
func RankOf(pageType *string, declared map[string]bool) Rank {
	if pageType == nil {
		return RankUnknown
	}
	switch t := *pageType; {
	case coreTypes()[t]:
		return RankCore
	case catalogueTypes()[t]:
		return RankCatalogue
	case originTypes()[t]:
		return RankOrigin
	case declared[t]:
		return RankDeclared
	}
	return RankUnknown
}

// AliasTarget is `alias_target`: the catalogue name a known misspelling of
// the corpus stands for. The table grows with every migration and lives
// here because it is knowledge about the corpus, not a procedure.
func AliasTarget(pageType string) (string, bool) {
	target, ok := map[string]string{
		"Design Decision":       "Decision",
		"Architecture Decision": "Decision",
	}[pageType]
	return target, ok
}
