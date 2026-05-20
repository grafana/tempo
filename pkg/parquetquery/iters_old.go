package parquetquery

/*
// LeftJoinIterator joins two or more iterators for matches at the given definition level.
// The first set of required iterators must all produce matching results. The second set
// of optional iterators are collected if they also match.
// TODO - This should technically obsolete the JoinIterator.
type LeftJoinIterator struct {
	definitionLevel              int
	required, optional           []Iterator
	peeksRequired, peeksOptional []*IteratorResult
	pred                         GroupPredicate
	pool                         *ResultPool
	at                           *IteratorResult

	// Unused
	collector                Collector
	name                     string
	collectedThroughRequired []RowNumber
	collectedThroughOptional []RowNumber
	defLevelsRequired        []int
	defLevelsOptional        []int
	paramsRequired           []any
	paramsOptional           []any
}

var _ Iterator = (*LeftJoinIterator)(nil)

func NewLeftJoinIterator(definitionLevel int, required, optional []Iterator, pred GroupPredicate, opts ...LeftJoinIteratorOption) (*LeftJoinIterator, error) {
	// No query should ever result in a left-join with no required iterators.
	// If this happens, it's a bug in the iter building code.
	// LeftJoinIterator is not designed to handle this case and will loop forever.
	if len(required) == 0 {
		return nil, fmt.Errorf("left join iterator requires at least one required iterator")
	}

	j := &LeftJoinIterator{
		definitionLevel: definitionLevel,
		required:        required,
		optional:        optional,
		peeksRequired:   make([]*IteratorResult, len(required)),
		peeksOptional:   make([]*IteratorResult, len(optional)),
		pred:            pred,
		pool:            DefaultPool,
	}

	for _, opt := range opts {
		opt.applyToLeftJoinIterator(j)
	}

	j.at = j.pool.Get()

	return j, nil
}

func (j *LeftJoinIterator) String() string {
	srequired := "required: "
	for _, r := range j.required {
		srequired += "\n\t" + util.TabOut(r)
	}
	soptional := "optional: "
	for _, o := range j.optional {
		soptional += "\n\t" + util.TabOut(o)
	}
	return fmt.Sprintf("LeftJoinIterator: %d: %s\n%s\n%s", j.definitionLevel, j.pred, srequired, soptional)
}

func (j *LeftJoinIterator) Next() (*IteratorResult, error) {
outer:
	for {
		// This loop is doing two things:
		// On first-pass peek each required iter and ensure it has
		// at least one result.  If any iter has no results we can
		// exit early without processing the remaining data in the others.
		// On subsequent passes the first iter is never nil except
		// when everything is fully exhausted. We check once more
		// and then exit.
		if j.peeksRequired[0] == nil {
			for i := range j.peeksRequired {
				res, err := j.peek(i)
				if err != nil {
					return nil, err
				}
				if res == nil {
					return nil, nil
				}
			}
		}

		// The first iter is pointing at the next candidate row. Proceed through iters 2 to N looking
		// for matches.
		for iterNum := 1; iterNum < len(j.required); iterNum++ {
			err := j.seek(iterNum, j.peeksRequired[0].RowNumber, j.definitionLevel)
			if err != nil {
				return nil, err
			}

			if j.peeksRequired[iterNum] == nil {
				// This iterator is exhausted no more joins possible.
				return nil, nil
			}

			if CompareRowNumbers(j.definitionLevel, j.peeksRequired[iterNum].RowNumber, j.peeksRequired[0].RowNumber) == 1 {
				// This iterator has a higher row number than all previous iterators.  That means it might have
				// a higher filtering power, swap it to the top and restart the loop.
				j.required[0], j.required[iterNum] = j.required[iterNum], j.required[0]
				j.peeksRequired[0], j.peeksRequired[iterNum] = j.peeksRequired[iterNum], j.peeksRequired[0]
				continue outer
			}
		}

		// All iterators pointing at same row
		// Get the data
		result, err := j.collect(j.peeksRequired[0].RowNumber)
		if err != nil {
			return nil, err
		}

		// Keep group?
		if j.pred == nil || j.pred.KeepGroup(result) {
			// Yes
			return result, nil
		}
	}
}

func (j *LeftJoinIterator) SeekTo(t RowNumber, d int) (*IteratorResult, error) {
	done, err := j.seekAllRequired(t, d)
	if err != nil {
		return nil, err
	}

	if done {
		// A required iterator is exhausted, no reason to seek the remaining
		return nil, nil
	}

	err = j.seekAllOptional(t, d)
	if err != nil {
		return nil, err
	}

	return j.Next()
}

func (j *LeftJoinIterator) seek(iterNum int, t RowNumber, d int) (err error) {
	if j.peeksRequired[iterNum] == nil || CompareRowNumbers(d, j.peeksRequired[iterNum].RowNumber, t) == -1 {
		// Release peek if present
		// These results have been collected but never returned upstream,
		// so we know it is safe to release them.
		if j.peeksRequired[iterNum] != nil {
			j.peeksRequired[iterNum].Release()
		}

		j.peeksRequired[iterNum], err = j.required[iterNum].SeekTo(t, d)
		if err != nil {
			return
		}
	}
	return nil
}

func (j *LeftJoinIterator) seekAllRequired(t RowNumber, d int) (done bool, err error) {
	for iterNum, iter := range j.required {
		if j.peeksRequired[iterNum] == nil || CompareRowNumbers(d, j.peeksRequired[iterNum].RowNumber, t) == -1 {

			// Release peek if present
			// These results have been collected but never returned upstream,
			// so we know it is safe to release them.
			if j.peeksRequired[iterNum] != nil {
				j.peeksRequired[iterNum].Release()
			}

			j.peeksRequired[iterNum], err = iter.SeekTo(t, d)
			if err != nil {
				return
			}
			if j.peeksRequired[iterNum] == nil {
				// A required iterator is exhausted, no reason to seek the remaining
				return true, nil
			}
		}
	}
	return
}

func (j *LeftJoinIterator) seekAllOptional(t RowNumber, d int) (err error) {
	for iterNum, iter := range j.optional {
		if j.peeksOptional[iterNum] == nil || CompareRowNumbers(d, j.peeksOptional[iterNum].RowNumber, t) == -1 {
			j.peeksOptional[iterNum], err = iter.SeekTo(t, d)
			if err != nil {
				return
			}
		}
	}
	return nil
}

func (j *LeftJoinIterator) peek(iterNum int) (*IteratorResult, error) {
	var err error
	if j.peeksRequired[iterNum] == nil {
		j.peeksRequired[iterNum], err = j.required[iterNum].Next()
		if err != nil {
			return nil, err
		}
	}
	return j.peeksRequired[iterNum], nil
}

// Collect data from the given iterators until they point at
// the next row (according to the configured definition level)
// or are exhausted.
func (j *LeftJoinIterator) collect(rowNumber RowNumber) (*IteratorResult, error) {
	var err error

	result := j.at
	result.Reset()
	result.RowNumber = rowNumber

	// Collect is only called after we have found a match among all
	// required iterators, therefore we only need to seek the optional ones to same location.
	if len(j.optional) > 0 {
		err = j.seekAllOptional(rowNumber, j.definitionLevel)
		if err != nil {
			return nil, err
		}
	}

	err = j.collectInternal(rowNumber, result, j.required, j.peeksRequired)
	if err != nil {
		return nil, err
	}

	if len(j.optional) > 0 {
		err = j.collectInternal(rowNumber, result, j.optional, j.peeksOptional)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (j *LeftJoinIterator) collectInternal(rowNumber RowNumber, result *IteratorResult, iters []Iterator, peeks []*IteratorResult) (err error) {
iters:
	for i := range iters {
		// Collect matches
		for peeks[i] != nil {
			// Interned version of EqualRowNumber
			// Compare in reverse order because most row number activity
			// occurs at the deeper definition levels.
			for k := j.definitionLevel; k >= 0; k-- {
				if peeks[i].RowNumber[k] != rowNumber[k] {
					continue iters
				}
			}

			result.Append(peeks[i])
			peeks[i], err = iters[i].Next()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (j *LeftJoinIterator) Close() {
	for _, i := range j.required {
		i.Close()
	}
	for _, i := range j.optional {
		i.Close()
	}
	j.pool.Release(j.at)
}
*/
