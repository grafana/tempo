package drain

import (
	"slices"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/require"
)

func TestDrain_PruneTreeClearsOldBranches(t *testing.T) {
	t.Parallel()

	inputLines := []string{
		"test test test 123",
		"test test test 456",
		"test test test 789",
		"test test test 101",
		"my name is 104",
		"my name is 105",
		"my name is 106",
		"my name is 107",
	}

	drain := New("test-tenant", DefaultConfig())

	for _, line := range inputLines {
		drain.Train(line)
	}

	require.Len(t, drain.Clusters(), 2)
	require.Equal(t, 16, countNodes(drain.rootNode))

	// Delete a cluster manually
	drain.idToCluster.Remove(1)
	require.Len(t, drain.Clusters(), 1)
	require.Equal(t, 16, countNodes(drain.rootNode), "expected same number of nodes before pruning")

	drain.Prune()
	require.Len(t, drain.Clusters(), 1)
	require.Equal(t, 9, countNodes(drain.rootNode), "expected fewer nodes after pruning")
}

func countNodes(node *Node) int {
	total := 1
	for _, child := range node.keyToChildNode {
		total += countNodes(child)
	}
	return total
}

func TestDrain_Train_BasicPatternDetection(t *testing.T) {
	t.Parallel()

	drain := New("test-tenant", DefaultConfig())

	// Train with similar span names that should form a pattern
	spanNames := []string{
		"GET /api/users/123",
		"GET /api/users/456",
		"GET /api/users/789",
		"GET /api/users/101",
	}

	var clusters []*LogCluster
	for _, spanName := range spanNames {
		cluster := drain.Train(spanName)
		require.NotNil(t, cluster)
		clusters = append(clusters, cluster)
	}

	// After training multiple similar patterns, they should cluster together
	// All clusters should have the same pattern (with parameter markers)
	uniquePatterns := make(map[string]bool)
	for _, cluster := range clusters {
		pattern := cluster.String()
		uniquePatterns[pattern] = true
	}

	// Should have fewer unique patterns than input span names (clustering occurred)
	require.Less(t, len(uniquePatterns), len(spanNames), "patterns should be clustered")

	// Verify pattern contains parameter marker
	for pattern := range uniquePatterns {
		require.Contains(t, pattern, "<_>", "pattern should contain parameter marker")
	}
}

func TestDrain_Train_MultipleClusters(t *testing.T) {
	t.Parallel()

	drain := New("test-tenant", DefaultConfig())

	// Train with different patterns
	spanNames := []string{
		"GET /api/users/123",
		"GET /api/users/456",
		"POST /api/posts/789",
		"POST /api/posts/101",
		"PUT /api/comments/202",
		"DELETE /api/comments/303",
	}

	for _, spanName := range spanNames {
		cluster := drain.Train(spanName)
		require.NotNil(t, cluster)
	}

	allClusters := drain.Clusters()
	// Should have multiple distinct clusters (at least 2-3 different patterns)
	require.GreaterOrEqual(t, len(allClusters), 2, "should have multiple clusters")

	// Verify clusters have different patterns
	patterns := make(map[string]bool)
	for _, cluster := range allClusters {
		pattern := cluster.String()
		patterns[pattern] = true
	}
	require.GreaterOrEqual(t, len(patterns), 2, "should have multiple distinct patterns")
}

func TestDrain_Train_MinTokensEnforcement(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.MinTokens = 3 // Need at least 3 tokens
	drain := New("test-tenant", config)

	// Span name that will tokenize to less than MinTokens
	// Single character will tokenize to ["a", "<END>"] which is 2 tokens < 3
	cluster := drain.Train("a")
	require.Nil(t, cluster, "should return nil for span with too few tokens")

	// Valid span name should work
	cluster = drain.Train("GET /api/users")
	require.NotNil(t, cluster, "should accept valid span name")
}

func TestDrain_Train_MaxTokensEnforcement(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.MaxTokens = 3 // Very small limit for testing (tokenizer produces many tokens)
	drain := New("test-tenant", config)

	// Create a span name that will exceed MaxTokens
	// "GET /api/users" tokenizes to ["GET", " ", "/", "api", "/", "users", "<END>"] = 7 tokens
	longSpanName := "GET /api/users/123/posts/456/comments/789/tags/101/items/202/details/303"
	cluster := drain.Train(longSpanName)
	require.Nil(t, cluster, "should return nil for span with too many tokens")
}

func TestDrain_Train_EmptyString(t *testing.T) {
	t.Parallel()

	drain := New("test-tenant", DefaultConfig())

	// Empty string should return nil
	cluster := drain.Train("")
	require.Nil(t, cluster, "should return nil for empty string")
}

func TestDrain_Train_CacheEvictionIntegration(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.MaxClusters = 3 // Small cache size to trigger evictions
	config.StaleClusterAge = 1 * time.Hour
	drain := New("test-tenant", config)

	// Fill cache beyond max size
	spanNames := []string{
		"GET /api/users/1",
		"GET /api/users/2",
		"GET /api/posts/3",
		"GET /api/posts/4",
		"GET /api/comments/5",
	}

	for _, spanName := range spanNames {
		cluster := drain.Train(spanName)
		require.NotNil(t, cluster)
	}

	// After filling cache, some clusters may have been evicted
	allClusters := drain.Clusters()
	require.LessOrEqual(t, len(allClusters), config.MaxClusters, "cache should respect max size")

	// Prune should clean up references to evicted clusters
	drain.Prune()

	// After pruning, clusters count should still be <= max
	allClustersAfterPrune := drain.Clusters()
	require.LessOrEqual(t, len(allClustersAfterPrune), config.MaxClusters)
}

func TestDrain_Train_PatternEmergence(t *testing.T) {
	t.Parallel()

	drain := New("test-tenant", DefaultConfig())

	// First span - creates initial cluster
	cluster1 := drain.Train("GET /api/users/123")
	require.NotNil(t, cluster1)
	pattern1 := cluster1.String()

	// Second similar span - should match and update pattern
	cluster2 := drain.Train("GET /api/users/456")
	require.NotNil(t, cluster2)

	pattern2 := cluster2.String()
	require.Contains(t, pattern2, "<_>", "pattern should emerge with parameter marker")
	require.NotEqual(t, pattern1, pattern2, "pattern should change after merging")
}

func TestDrain_TrainReleasesInputReferences(t *testing.T) {
	d := New(t.Name(), DefaultConfig())
	d.Train("GET /api/users/123/children/456")
	d.Train("GET /api")

	for _, token := range d.tokenBuffer[:cap(d.tokenBuffer)] {
		require.Empty(t, token)
	}
	tokenizer := d.tokenizer.(*defaultTokenizer)
	require.Empty(t, tokenizer.lexer.input)
	require.Empty(t, tokenizer.lexer.token)
}

func TestDrain_ExactIndexHit(t *testing.T) {
	d, lines, clusters := newCollapsedDrain(t, exactMatchIndexThreshold)
	target := clusters[3]
	leaf := collapsedLeaf(d)

	require.NotNil(t, leaf.exactClusters)
	require.Same(t, target, d.findExactCluster(leaf, lines[3]))

	leaf.clusterIDs = slices.DeleteFunc(leaf.clusterIDs, func(id int) bool {
		return id == target.id
	})
	size := target.Size
	require.Same(t, target, d.Train(lines[3]))
	require.Equal(t, size+1, target.Size)
}

func TestDrain_ExactIndexInvalidatedOnGeneralization(t *testing.T) {
	d := New(t.Name(), DefaultConfig())
	cluster := d.Train("GET /api/users/123")
	leaf := findClusterNode(d.rootNode, cluster.id)
	require.NotNil(t, leaf)
	leaf.exactClusters = make(map[uint64]int)
	d.indexExactCluster(leaf, cluster, "")
	location := d.exactLocations[cluster.id]

	require.Same(t, cluster, d.Train("GET /api/users/456"))
	require.False(t, cluster.isExact())
	require.NotContains(t, d.exactLocations, cluster.id)
	require.NotContains(t, leaf.exactClusters, location.hash)
}

func TestDrain_ExactIndexHashCollisionFallsBack(t *testing.T) {
	d, lines, clusters := newCollapsedDrain(t, exactMatchIndexThreshold)
	target := clusters[2]
	other := clusters[7]
	leaf := collapsedLeaf(d)
	location := d.exactLocations[target.id]
	leaf.exactClusters[location.hash] = other.id

	require.Nil(t, d.findExactCluster(leaf, lines[2]))
	require.Same(t, target, d.Train(lines[2]))
}

func TestDrain_CompactsEvictedClusterIDs(t *testing.T) {
	d, _, clusters := newCollapsedDrain(t, exactMatchIndexThreshold)
	leaf := collapsedLeaf(d)
	d.idToCluster.Remove(clusters[2].id)
	d.idToCluster.Remove(clusters[9].id)

	cluster := d.Train(collapsedSpanName(100))
	expected := make([]int, 0, len(clusters)-1)
	for i, existing := range clusters {
		if i != 2 && i != 9 {
			expected = append(expected, existing.id)
		}
	}
	expected = append(expected, cluster.id)
	require.Equal(t, expected, leaf.clusterIDs)
}

func TestDrain_TrainCleansDeletedExactIndex(t *testing.T) {
	d, _, clusters := newCollapsedDrain(t, exactMatchIndexThreshold)
	cluster := clusters[4]
	d.idToCluster.Remove(cluster.id)

	d.Train(collapsedSpanName(100))

	require.NotContains(t, d.exactLocations, cluster.id)
}

func TestDrain_PruneCleansExactIndex(t *testing.T) {
	d, _, clusters := newCollapsedDrain(t, exactMatchIndexThreshold)
	cluster := clusters[4]
	location := d.exactLocations[cluster.id]
	d.idToCluster.Remove(cluster.id)

	d.Prune()

	require.NotContains(t, location.node.exactClusters, location.hash)
	require.NotContains(t, d.exactLocations, cluster.id)
}

func TestDrain_DeleteCleansExactIndex(t *testing.T) {
	d, _, clusters := newCollapsedDrain(t, exactMatchIndexThreshold)
	cluster := clusters[4]
	location := d.exactLocations[cluster.id]

	d.Delete(cluster)

	require.NotContains(t, location.node.exactClusters, location.hash)
	require.NotContains(t, d.exactLocations, cluster.id)
	require.Nil(t, d.idToCluster.GetQuietly(cluster.id))
}

func TestDrain_ExactIndexSizeIsBounded(t *testing.T) {
	const maxClusters = 16
	cfg := DefaultConfig()
	cfg.LogClusterDepth = 3
	cfg.SimTh = 1
	cfg.MaxClusters = maxClusters
	cfg.StaleClusterAge = time.Hour
	d := New(t.Name(), cfg)

	for i := range 1000 {
		d.Train(collapsedSpanName(i))
	}

	require.LessOrEqual(t, len(d.exactLocations), maxClusters)
	require.LessOrEqual(t, len(collapsedLeaf(d).exactClusters), maxClusters)
}

func TestDrain_AutomaticEvictionCleansExactIndex(t *testing.T) {
	const maxClusters = 16
	cfg := DefaultConfig()
	cfg.LogClusterDepth = 3
	cfg.SimTh = 1
	cfg.MaxClusters = maxClusters
	cfg.StaleClusterAge = time.Hour
	d := New(t.Name(), cfg)

	for i := range 100 {
		d.Train(collapsedSpanName(i))
	}
	d.idToCluster.cache.CleanUp()
	d.removeDeletedExactClusters()

	require.LessOrEqual(t, len(d.exactLocations), maxClusters)
	for clusterID := range d.exactLocations {
		require.NotNil(t, d.idToCluster.GetQuietly(clusterID))
	}
}

func TestDrain_EmptyExactIndexDoesNotRebuild(t *testing.T) {
	d, _, clusters := newCollapsedDrain(t, exactMatchIndexThreshold)
	leaf := collapsedLeaf(d)
	for _, cluster := range clusters {
		d.removeExactCluster(cluster.id)
	}
	require.Empty(t, d.exactLocations)
	require.True(t, leaf.exactIndexBuilt)

	d.Prune()
	require.Nil(t, leaf.exactClusters)
	require.True(t, leaf.exactIndexBuilt)

	cluster := d.Train(collapsedSpanName(100))
	require.Equal(t, map[int]exactLocation{
		cluster.id: {node: leaf, hash: xxhash.Sum64String(cluster.String())},
	}, d.exactLocations)
}

func newCollapsedDrain(t testing.TB, population int) (*Drain, []string, []*LogCluster) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.LogClusterDepth = 3
	cfg.SimTh = 1
	cfg.MaxClusters = population + 1
	cfg.StaleClusterAge = time.Hour
	d := New(t.Name(), cfg)
	lines := make([]string, population)
	clusters := make([]*LogCluster, population)
	for i := range population {
		lines[i] = collapsedSpanName(i)
		clusters[i] = d.Train(lines[i])
	}
	return d, lines, clusters
}

func collapsedLeaf(d *Drain) *Node {
	return findPopulatedLeaf(d.rootNode)
}

func findPopulatedLeaf(node *Node) *Node {
	if len(node.clusterIDs) > 0 {
		return node
	}
	for _, child := range node.keyToChildNode {
		if found := findPopulatedLeaf(child); found != nil {
			return found
		}
	}
	return nil
}

func findClusterNode(node *Node, clusterID int) *Node {
	if slices.Contains(node.clusterIDs, clusterID) {
		return node
	}
	for _, child := range node.keyToChildNode {
		if found := findClusterNode(child, clusterID); found != nil {
			return found
		}
	}
	return nil
}

func collapsedSpanName(i int) string {
	return "g" + alphabetic(i) + "-" + "g" + alphabetic(i+17576)
}

func alphabetic(i int) string {
	var buf [16]byte
	pos := len(buf)
	for {
		pos--
		buf[pos] = byte('a' + i%26)
		i /= 26
		if i == 0 {
			return string(buf[pos:])
		}
	}
}
