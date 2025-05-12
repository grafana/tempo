package exemplardist

import (
    "math"
    "sort"
)

// DistributeExemplars ensures exemplars are evenly distributed across a time range
// rather than being clustered on one side
//
// Parameters:
// - exemplars: The timestamps of exemplars in milliseconds
// - startMs: The start time of the range in milliseconds
// - endMs: The end time of the range in milliseconds
// - maxExemplars: Maximum number of exemplars to return
//
// Returns:
// - Indices of the selected exemplars in the original array
func DistributeExemplars(
    exemplars []uint64,
    startMs, endMs uint64,
    maxExemplars int,
) []int {
    // If we have fewer exemplars than the max, return all of them
    if len(exemplars) <= maxExemplars {
        indices := make([]int, len(exemplars))
        for i := range exemplars {
            indices[i] = i
        }
        return indices
    }

    // Ensure start is before end
    if startMs >= endMs {
        // Return the first maxExemplars
        indices := make([]int, maxExemplars)
        for i := 0; i < maxExemplars; i++ {
            indices[i] = i
        }
        return indices
    }

    // Calculate time range
    timeRange := endMs - startMs

    // Create buckets across the time range
    bucketCount := maxExemplars
    buckets := make([][]int, bucketCount)
    for i := range buckets {
        buckets[i] = []int{}
    }

    // Assign each exemplar to a bucket based on its timestamp
    for i, ts := range exemplars {
        // Skip exemplars outside the visible range
        if ts < startMs || ts > endMs {
            continue
        }

        // Calculate which bucket this exemplar belongs to
        bucketSize := float64(timeRange) / float64(bucketCount)
        bucketIndex := int(math.Min(
            float64(bucketCount-1),
            math.Floor(float64(ts-startMs)/bucketSize),
        ))

        buckets[bucketIndex] = append(buckets[bucketIndex], i)
    }

    // Select one exemplar from each bucket, preferring the middle one
    selectedIndices := []int{}
    for _, bucket := range buckets {
        if len(bucket) > 0 {
            // If bucket has multiple exemplars, pick the one closest to the middle
            middleIndex := len(bucket) / 2
            selectedIndices = append(selectedIndices, bucket[middleIndex])
        }
    }

    // If we still have room and some buckets were empty, fill with remaining exemplars
    if len(selectedIndices) < maxExemplars {
        extraNeeded := maxExemplars - len(selectedIndices)
        extraIndices := []int{}

        // Collect extra exemplars from buckets with more than one item
        for _, bucket := range buckets {
            if len(bucket) > 1 {
                middleIndex := len(bucket) / 2
                for i, idx := range bucket {
                    if i != middleIndex {
                        extraIndices = append(extraIndices, idx)
                    }
                }
            }
        }

        // Sort extra exemplars by timestamp
        sort.Slice(extraIndices, func(i, j int) bool {
            return exemplars[extraIndices[i]] < exemplars[extraIndices[j]]
        })

        // Add extra exemplars, evenly distributed
        if len(extraIndices) > 0 {
            step := int(math.Max(1, math.Floor(float64(len(extraIndices))/float64(extraNeeded))))
            for i := 0; i < extraNeeded && i*step < len(extraIndices); i++ {
                selectedIndices = append(selectedIndices, extraIndices[i*step])
            }
        }
    }

    // Sort selected indices by timestamp
    sort.Slice(selectedIndices, func(i, j int) bool {
        return exemplars[selectedIndices[i]] < exemplars[selectedIndices[j]]
    })

    return selectedIndices
}
