# Exemplar Distribution

This package provides an algorithm to evenly distribute exemplars across a time range, solving the issue where TraceQL metrics exemplars cluster on one side of the visualization instead of being distributed homogeneously.

## Problem

TraceQL metrics exemplars are not distributed homogeneously across time series but instead cluster toward one side (either left or right). This creates a biased view of the data and makes it harder for users to get representative exemplars across the entire time range.

## Solution

The `DistributeExemplars` function implements a bucketing algorithm that ensures exemplars are evenly distributed across the time range:

1. Divides the visible time range into equal-sized buckets (number of buckets = max exemplars to display)
2. Assigns each exemplar to its appropriate time bucket
3. Selects one exemplar from each bucket (preferably from the middle for representativeness)
4. If some buckets are empty, fills remaining slots from densely populated buckets while maintaining distribution

## Usage

```go
// Example usage
import "github.com/grafana/tempo/pkg/exemplardist"

// Collect timestamps from your exemplars
timestamps := make([]uint64, len(exemplars))
for i, exemplar := range exemplars {
    timestamps[i] = exemplar.TimestampMs
}

// Apply distribution algorithm
selectedIndices := exemplardist.DistributeExemplars(
    timestamps,
    startTimeMs,
    endTimeMs,
    maxExemplars,
)

// Use the selected indices to pick exemplars from your original collection
distributedExemplars := make([]MyExemplarType, len(selectedIndices))
for i, idx := range selectedIndices {
    distributedExemplars[i] = exemplars[idx]
}
