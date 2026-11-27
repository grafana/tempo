package base

// IteratorState is a SyncIterator's position and what it keeps: the row it is at, the
// target of a seek, its filter and sampler, and the filter's decisions. The
// iterator owns it; its page reader advances it over every slot it reads.
type IteratorState struct {
	Row    RowNumber // the row number of the last slot read
	MaxDef int       // the iterator's max definition level

	// Set for a seek: the slots before SeekTo, compared up to definition level SeekDef,
	// are skipped without being filtered.
	SeekTo  RowNumber
	SeekDef int

	Filter  Predicate // nil keeps every value
	Sampler Sampler   // nil keeps every value

	Stats  PredicateStats // the filter's chunk, page and value decisions
	Tenant string         // labels Stats when they're exported, from the iterator's ctx
}

// BeforeSeekTarget reports whether Row is before SeekTo.
func (s *IteratorState) BeforeSeekTarget() bool {
	for i := 0; i <= s.SeekDef; i++ {
		if s.Row[i] < s.SeekTo[i] {
			return true
		}
		if s.Row[i] > s.SeekTo[i] {
			return false
		}
	}
	return false
}
