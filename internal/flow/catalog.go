package flow

import (
	"fmt"
	"slices"
)

// Catalog is the Registry every loader and every runner shares: the blocks and
// predicates this build knows. It takes what it holds at construction and has
// no way to add later, so nothing can widen a running flow's vocabulary.
type Catalog struct {
	blocks     map[string]Block
	predicates map[string]Predicate
}

// Catalog stays a Registry. A change to the interface that Catalog stops
// matching breaks this package, where the interface lives, and not a package
// far from it.
var _ Registry = (*Catalog)(nil)

// NewCatalog gathers blocks by their own Kind and refuses a kind twice: a
// second block under a known name would shadow the first silently, and which
// one wins would depend on the order somebody wrote a slice literal in. A nil
// predicate is refused too: it would be found like any other and panic inside
// the first condition that names it.
func NewCatalog(blocks []Block, predicates map[string]Predicate) (*Catalog, error) {
	made := &Catalog{blocks: make(map[string]Block, len(blocks)), predicates: make(map[string]Predicate, len(predicates))}
	for _, block := range blocks {
		if _, taken := made.blocks[block.Kind()]; taken {
			return nil, fmt.Errorf("two blocks claim kind %q", block.Kind())
		}
		made.blocks[block.Kind()] = block
	}
	// In name order, so that of two nil predicates the same one is named every
	// time.
	names := make([]string, 0, len(predicates))
	for name := range predicates {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if predicates[name] == nil {
			return nil, fmt.Errorf("predicate %q is nil", name)
		}
		made.predicates[name] = predicates[name]
	}
	return made, nil
}

// Block finds the block for a node kind.
func (c *Catalog) Block(kind string) (Block, bool) {
	block, ok := c.blocks[kind]
	return block, ok
}

// Predicate finds a predicate by name.
func (c *Catalog) Predicate(name string) (Predicate, bool) {
	predicate, ok := c.predicates[name]
	return predicate, ok
}

// Kinds are the registered node kinds, sorted, as a load message lists them.
func (c *Catalog) Kinds() []string {
	kinds := make([]string, 0, len(c.blocks))
	for kind := range c.blocks {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	return kinds
}

// Predicates is what expr.ParseCondition wants: a copy, so that a caller
// writing into it cannot widen what every later flow may name.
func (c *Catalog) Predicates() map[string]Predicate {
	copied := make(map[string]Predicate, len(c.predicates))
	for name, predicate := range c.predicates {
		copied[name] = predicate
	}
	return copied
}
