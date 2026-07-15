package parquetquery

// chunkResetter is a TEMPORARY hook: regex/substring predicates memoize matches per
// column chunk and clear that cache at each chunk boundary; composites cascade.
// Removed once dictionary-index pushdown lands and predicate memoization goes away.
type chunkResetter interface {
	resetForChunk()
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

func (p *InstrumentedPredicate) resetForChunk() {
	if r, ok := p.Pred.(chunkResetter); ok {
		r.resetForChunk()
	}
}
