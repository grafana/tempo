package parquetquery

// chunkResetter is a TEMPORARY hook: regex/substring predicates memoize matches per
// column chunk and clear that cache at each chunk boundary; composites cascade.
// Removed once dictionary-index pushdown lands and predicate memoization goes away.
type chunkResetter interface {
	resetForChunk()
}

// resetFilter is TEMPORARY, removed with chunkResetter: it clears the filter's
// memoization at a chunk boundary. It runs before the dictionary scan, so the matches
// that scan memoizes serve the chunk's rows too. c.filter is the full tree, so resets
// cascade.
func (c *SyncIterator) resetFilter() {
	if r, ok := c.filter.(chunkResetter); ok {
		r.resetForChunk()
	}
}

func (p *regexPredicate) resetForChunk() { p.matcher.Reset() }

func (p *SubstringPredicate) resetForChunk() {
	p.matches = make(map[string]bool, len(p.matches))
}

func (p *OrPredicate) resetForChunk() {
	for _, sub := range p.preds {
		if r, ok := sub.(chunkResetter); ok {
			r.resetForChunk()
		}
	}
}
