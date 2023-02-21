// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package quantile

import (
	"sort"
	"unsafe"
)

var _ memSized = (*sparseStore)(nil)

type sparseStore struct {
	bins  binList
	count int
}

// Cols returns an array of k and n.
func (s *sparseStore) Cols() (k []int32, n []uint32) {
	if len(s.bins) == 0 {
		return
	}

	k = make([]int32, len(s.bins))
	n = make([]uint32, len(s.bins))

	// TODO: do this better.
	for i, b := range s.bins {
		k[i] = int32(b.k)
		n[i] = uint32(b.n)
	}

	return
}

// MemSize returns memory use in bytes:
//
//	used: uses len(bins)
//	allocated: uses cap(bins)
func (s *sparseStore) MemSize() (used, allocated int) {
	const (
		binSize   = int(unsafe.Sizeof(bin{}))
		storeSize = int(unsafe.Sizeof(sparseStore{}))
	)
	// cap is used instead of len because an improved algorithm would take advantage
	// of the unused space after a slice is resized.
	used = storeSize + (len(s.bins) * binSize)
	allocated = storeSize + (cap(s.bins) * binSize)
	return
}

// trimLeft ensures that len(a) <= maxBucketCap. We set maxBucketCap rather high
// by default to avoid trimming as much as possible.
func trimLeft(a []bin, maxBucketCap int) []bin {
	// XXX:
	// 1. Work through overflow cause
	// 2. CompressMode enum

	// TODO: Research alternate compression methods
	//
	// (1) Remove closest buckets
	// (2) re-gamma (kinda like hdr histogram)
	if maxBucketCap == 0 || len(a) <= maxBucketCap {
		return a
	}

	var (
		nRemove = len(a) - maxBucketCap

		missing  int
		overflow []bin
	)

	// TODO|PROD: Benchmark a better overflow scheme.
	// In theory, if we always have the smaller overflow in the lower bucket, we
	// can guarantee that only 1 extra bin is needed for overflow.
	// For example:

	// fmt = (<k>:<n>[ <k>:<num overflow>])
	//                               NEW        CURRENT
	// 1) (0:1) + (0:max*2)       = (0:1 0:2)  (0:1 0:max 0:max)
	// 2) (0:1 0:max) + (0:max-1) = (0:0 0:2)  (0:max 0:max)
	for i := 0; i < nRemove; i++ {
		missing += int(a[i].n)

		if missing > maxBinWidth {
			overflow = append(overflow, bin{
				k: a[i].k,
				n: maxBinWidth,
			})

			missing -= maxBinWidth
		}
	}

	missing = a[nRemove].incrSafe(missing)
	if missing > 0 {
		overflow = appendSafe(overflow, a[nRemove].k, missing)
	}

	copy(a, overflow)
	copy(a[len(overflow):], a[nRemove:])

	return a[:maxBucketCap+len(overflow)]
}

func (s *sparseStore) merge(c *Config, o *sparseStore) {
	// TODO|PERF: Compare blocky merge with other methods.
	// TODO|PERF: We have essentially unlimited tmp space, can we merge into a
	// dense store and then copy back to the sparse version?
	s.count += o.count
	tmp := getBinList()[:0]

	sIdx := 0
	for _, ob := range o.bins {

		for sIdx < s.bins.Len() && s.bins[sIdx].k < ob.k {
			tmp = append(tmp, s.bins[sIdx])
			sIdx++
		}

		// done with s
		switch {
		case sIdx >= s.bins.Len(), s.bins[sIdx].k > ob.k:
			tmp = append(tmp, ob)
		case s.bins[sIdx].k == ob.k:
			n := int(ob.n) + int(s.bins[sIdx].n)
			tmp = appendSafe(tmp, ob.k, n)
			sIdx++
		}
	}
	tmp = append(tmp, s.bins[sIdx:]...)
	tmp = trimLeft(tmp, c.binLimit)
	s.bins = s.bins.ensureLen(len(tmp))
	copy(s.bins, tmp)
	putBinList(tmp)
}

func (s *sparseStore) insertCounts(c *Config, kcs []KeyCount) {

	// TODO|PERF: A custom uint16 sort should easily beat sort.Sort.
	// TODO|PERF: Would it be cheaper to sort float64s and then convert to keys?
	sort.Slice(kcs, func(i, j int) bool {
		return kcs[i].k < kcs[j].k
	})

	// TODO|PERF: Add a non-allocating fast path. When every key is already contained
	// in the sketch (and no overflow happens) we can just directly update.
	tmp := getBinList()

	var (
		sIdx, keyIdx int
	)

	for sIdx < len(s.bins) && keyIdx < len(kcs) {
		b := s.bins[sIdx]
		vk := kcs[keyIdx].k
		kn := int(kcs[keyIdx].n)

		switch {
		case b.k < vk:
			tmp = append(tmp, b)
			sIdx++
		case b.k > vk:
			// When vk[i] == vk[i+1] we need to make sure they go in the same bucket.
			tmp = appendSafe(tmp, vk, kn)
			s.count += kn
			keyIdx++
		default:
			tmp = appendSafe(tmp, b.k, int(b.n)+kn)
			s.count += kn
			sIdx++
			keyIdx++
		}
	}

	tmp = append(tmp, s.bins[sIdx:]...)

	for keyIdx < len(kcs) {
		kn := int(kcs[keyIdx].n)
		tmp = appendSafe(tmp, kcs[keyIdx].k, kn)
		s.count += kn
		keyIdx++
	}

	tmp = trimLeft(tmp, c.binLimit)

	// TODO|PERF: reallocate if cap(s.bins) >> len(s.bins)
	s.bins = s.bins.ensureLen(len(tmp))
	copy(s.bins, tmp)
	putBinList(tmp)
}

func (s *sparseStore) insert(c *Config, keys []Key) {
	s.count += len(keys)

	// TODO|PERF: A custom uint16 sort should easily beat sort.Sort.
	// TODO|PERF: Would it be cheaper to sort float64s and then convert to keys?
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	// TODO|PERF: Add a non-allocating fast path. When every key is already contained
	// in the sketch (and no overflow happens) we can just directly update.
	tmp := getBinList()

	var (
		sIdx, keyIdx int
	)

	for sIdx < len(s.bins) && keyIdx < len(keys) {
		b := s.bins[sIdx]
		vk := keys[keyIdx]

		switch {
		case b.k < vk:
			tmp = append(tmp, b)
			sIdx++
		case b.k > vk:
			// When vk[i] == vk[i+1] we need to make sure they go in the same bucket.
			kn := bufCountLeadingEqual(keys, keyIdx)
			tmp = appendSafe(tmp, vk, kn)
			keyIdx += kn
		default:
			kn := bufCountLeadingEqual(keys, keyIdx)
			tmp = appendSafe(tmp, b.k, int(b.n)+kn)
			sIdx++
			keyIdx += kn
		}
	}

	tmp = append(tmp, s.bins[sIdx:]...)

	for keyIdx < len(keys) {
		kn := bufCountLeadingEqual(keys, keyIdx)
		tmp = appendSafe(tmp, keys[keyIdx], kn)
		keyIdx += kn
	}

	tmp = trimLeft(tmp, c.binLimit)

	// TODO|PERF: reallocate if cap(s.bins) >> len(s.bins)
	s.bins = s.bins.ensureLen(len(tmp))
	copy(s.bins, tmp)
	putBinList(tmp)
}

// bufCountLeadingEqual returns the number of consecutive keys in a[i:] that equal a[i].
// given:
//
//	i = 0 1 2 3 4 5 6
//	a = 4 5 6 8 8 8 9
//
// bufCountLeadingEqual(a, 3) = 3, a[3] = 8
// bufCountLeadingEqual(a, 4) = 2, a[4] = 8
func bufCountLeadingEqual(a []Key, start int) int {
	if start == len(a)-1 {
		return 1
	}

	i := start
	for i < len(a) && a[i] == a[start] {
		i++
	}

	return i - start
}
