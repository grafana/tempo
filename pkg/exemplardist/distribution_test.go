package exemplardist

import (
    "math"
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func TestDistributeExemplars(t *testing.T) {
    // Create test data with right-skewed distribution (clustering to the right)
    timestamps := createSkewedTimestamps(100, 1000, 2000, "right")
    maxExemplars := 20

    // Apply our distribution algorithm
    indices := DistributeExemplars(timestamps, 1000, 2000, maxExemplars)

    // We should get the exact number of exemplars requested
    assert.Equal(t, maxExemplars, len(indices), "Should return exactly max exemplars")

    // Extract the distributed timestamps
    distributed := make([]uint64, len(indices))
    for i, idx := range indices {
        distributed[i] = timestamps[idx]
    }

    // Verify the distribution quality
    quality := measureDistributionQuality(distributed, 1000, 2000, 5)
    assert.True(t, quality.isUniform, "Distribution should be uniform")
    
    // The original distribution (taking first N exemplars) should be worse
    original := timestamps[:maxExemplars]
    originalQuality := measureDistributionQuality(original, 1000, 2000, 5)
    assert.Greater(t, originalQuality.coefficientOfVariation, quality.coefficientOfVariation,
        "Original distribution should have higher coefficient of variation")
}

func TestDistributeExemplarsWithFewExemplars(t *testing.T) {
    // Test with fewer exemplars than max
    timestamps := createUniformTimestamps(5, 1000, 2000)
    maxExemplars := 20

    // Apply algorithm
    indices := DistributeExemplars(timestamps, 1000, 2000, maxExemplars)

    // Should return all exemplars
    assert.Equal(t, len(timestamps), len(indices), "Should return all exemplars when fewer than max")
}

// Helper functions

// Create timestamps with uniform distribution
func createUniformTimestamps(count int, startMs, endMs uint64) []uint64 {
    timestamps := make([]uint64, count)
    timeRange := endMs - startMs

    for i := 0; i < count; i++ {
        timestamp := startMs + uint64(float64(i)*float64(timeRange)/float64(count-1))
        timestamps[i] = timestamp
    }

    return timestamps
}

// Create timestamps with skewed distribution
func createSkewedTimestamps(count int, startMs, endMs uint64, direction string) []uint64 {
    timestamps := make([]uint64, count)
    timeRange := endMs - startMs

    for i := 0; i < count; i++ {
        var factor float64
        ratio := float64(i) / float64(count-1)

        if direction == "right" {
            // Skew to the right (cluster at the end)
            factor = math.Pow(ratio, 2)
        } else {
            // Skew to the left (cluster at the beginning)
            factor = math.Sqrt(ratio)
        }

        timestamp := startMs + uint64(factor*float64(timeRange))
        timestamps[i] = timestamp
    }

    return timestamps
}

// Distribution quality metrics
type distributionQuality struct {
    bucketCounts           []int
    coefficientOfVariation float64
    isUniform              bool
}

// Measure the quality of the distribution
func measureDistributionQuality(timestamps []uint64, startMs, endMs uint64, bucketCount int) distributionQuality {
    timeRange := endMs - startMs
    bucketSize := float64(timeRange) / float64(bucketCount)

    // Count exemplars in each bucket
    bucketCounts := make([]int, bucketCount)
    for _, ts := range timestamps {
        bucketIndex := int(math.Min(
            float64(bucketCount-1),
            math.Floor(float64(ts-startMs)/bucketSize),
        ))
        bucketCounts[bucketIndex]++
    }

    // Calculate metrics
    totalExemplars := len(timestamps)
    expectedPerBucket := float64(totalExemplars) / float64(bucketCount)

    // Calculate variance
    variance := 0.0
    for _, count := range bucketCounts {
        variance += math.Pow(float64(count)-expectedPerBucket, 2)
    }
    variance /= float64(bucketCount)

    // Calculate standard deviation and coefficient of variation
    stdDeviation := math.Sqrt(variance)
    coefficientOfVariation := 0.0
    if expectedPerBucket > 0 {
        coefficientOfVariation = stdDeviation / expectedPerBucket
    }

    // A distribution is uniform if the coefficient of variation is low
    isUniform := coefficientOfVariation < 0.5

    return distributionQuality{
        bucketCounts:           bucketCounts,
        coefficientOfVariation: coefficientOfVariation,
        isUniform:              isUniform,
    }
}
