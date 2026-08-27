package drain

import (
	"slices"
	"strconv"
	"strings"
)

// Sixteen is a heuristic cutoff: below it, a linear leaf scan is cheaper than
// maintaining an exact-match index.
const exactMatchIndexThreshold = 16

// Drain is an implementation of the DRAIN algorithm, specialized for clustering
// span names in Tempo.
type Drain struct {
	config            *Config
	maxNodeDepth      int
	metrics           *tenantMetrics
	tokenizer         LineTokenizer
	tokenIsLikelyData IsDataHeuristic

	// The primary state of the Drain algorithm consists of a prefix tree and a
	// cache of clusters. Clusters are named "log clusters" to match the
	// original DRAIN algorithm, but here are used to cluster span names.
	clustersCounter int
	rootNode        *Node
	idToCluster     *logClusterCache

	// exactLocationByClusterID is the reverse index used to remove an exact
	// pattern from its leaf when the cluster is deleted or generalized.
	exactLocationByClusterID map[int]exactLocation

	// tokenBuffer is reused to avoid allocating a new slice for each call to Train.
	tokenBuffer []string
}

// New creates a new Drain instance.
func New(tenant string, config *Config) *Drain {
	if config.LogClusterDepth < 3 {
		config.LogClusterDepth = 3
	}

	metrics := metricsForTenant(tenant)
	d := &Drain{
		config:            config,
		maxNodeDepth:      config.LogClusterDepth - 2,
		metrics:           metrics,
		tokenizer:         &defaultTokenizer{},
		tokenIsLikelyData: defaultIsDataHeuristic,

		rootNode:                 newNode(),
		idToCluster:              newLogClusterCache(config.StaleClusterAge, config.MaxClusters, metrics.PatternsEvictedTotal, metrics.PatternsExpiredTotal),
		exactLocationByClusterID: make(map[int]exactLocation),
	}

	return d
}

type exactLocation struct {
	node    *Node
	pattern string
}

type Node struct {
	keyToChildNode map[string]*Node
	clusterIDs     []int

	// exactClusterIDByPattern avoids a linear cluster scan for a repeated
	// exact pattern after the normal DRAIN tree traversal reaches this leaf.
	exactClusterIDByPattern map[string]int
	// Distinguishes a drained index from a leaf that has never been large enough
	// to index, avoiding a full rebuild when another exact cluster is added.
	exactIndexBuilt bool
}

func newNode() *Node {
	return &Node{
		keyToChildNode: make(map[string]*Node),
		clusterIDs:     make([]int, 0),
	}
}

func (d *Drain) Clusters() []*LogCluster {
	return slices.Collect(d.idToCluster.Values())
}

func (d *Drain) Train(content string) *LogCluster {
	d.removeDeletedExactClusters()
	d.tokenBuffer = d.tokenizer.Tokenize(content, d.tokenBuffer)
	// Token strings alias content; clear them before retaining the slice for reuse.
	defer func() {
		clear(d.tokenBuffer)
		d.tokenBuffer = d.tokenBuffer[:0]
	}()

	if len(d.tokenBuffer) == 0 {
		return nil
	}

	if len(d.tokenBuffer) < d.config.MinTokens {
		d.metrics.LinesSkippedTooFewTokens.Inc()
		return nil
	}
	if len(d.tokenBuffer) > d.config.MaxTokens {
		d.metrics.LinesSkippedTooManyTokens.Inc()
		return nil
	}
	d.metrics.TokensPerLine.Observe(float64(len(d.tokenBuffer)))

	return d.train(content, d.tokenBuffer)
}

func (d *Drain) newCluster(tokens []string) *LogCluster {
	d.clustersCounter++
	return &LogCluster{
		Tokens:      cloneTokenStrings(tokens),
		id:          d.clustersCounter,
		Size:        1,
		Stringer:    d.tokenizer.Join,
		ParamString: d.config.ParamString,
	}
}

// Pack retained token bytes into one owned allocation so clusters never retain caller input.
func cloneTokenStrings(tokens []string) []string {
	var builder strings.Builder
	totalLen := 0
	for _, token := range tokens {
		totalLen += len(token)
	}
	builder.Grow(totalLen)
	for _, token := range tokens {
		builder.WriteString(token)
	}
	owned := builder.String()
	cloned := make([]string, len(tokens))
	offset := 0
	for i, token := range tokens {
		next := offset + len(token)
		cloned[i] = owned[offset:next]
		offset = next
	}
	return cloned
}

func (d *Drain) train(content string, tokens []string) *LogCluster {
	for {
		cluster := d.findMatchingClusterForTokens(content, tokens)
		if cluster == nil {
			cluster = d.newCluster(tokens)
			d.addClusterToRootNode(cluster)
			d.metrics.PatternsDetectedTotal.Inc()
			d.idToCluster.Put(cluster)
			return cluster
		}

		// Refresh access expiry only after selecting the best candidate. It may
		// expire between the quiet lookup and this refresh, in which case retry.
		cluster = d.idToCluster.Get(cluster.id)
		if cluster == nil {
			continue
		}
		if cluster.ingestTokens(tokens) {
			d.removeExactCluster(cluster.id)
		}
		return cluster
	}
}

func (d *Drain) removeDeletedExactClusters() {
	for _, clusterID := range d.idToCluster.TakeDeleted() {
		d.removeExactCluster(clusterID)
	}
}

// Prune removes old branches from the tree. We rely on the cache eviction
// algorithm to remove clusters from the cache, then this method will remove
// references to them in the tree.
func (d *Drain) Prune() {
	d.removeDeletedExactClusters()
	d.pruneTree(d.rootNode)
}

// pruneTree removes old branches from a node and its children. Nodes are pruned
// once they reference no valid clusters in either the scan list or exact index.
func (d *Drain) pruneTree(node *Node) int {
	for key, child := range node.keyToChildNode {
		if d.pruneTree(child) == 0 {
			delete(node.keyToChildNode, key)
		}
	}

	node.clusterIDs = slices.DeleteFunc(node.clusterIDs, d.idToCluster.NotExists)
	for pattern, clusterID := range node.exactClusterIDByPattern {
		cluster := d.idToCluster.GetQuietly(clusterID)
		if cluster == nil || !cluster.isExact() {
			delete(node.exactClusterIDByPattern, pattern)
			delete(d.exactLocationByClusterID, clusterID)
		}
	}
	if len(node.exactClusterIDByPattern) == 0 {
		node.exactClusterIDByPattern = nil
		if len(node.clusterIDs) < exactMatchIndexThreshold {
			node.exactIndexBuilt = false
		}
	}
	return len(node.keyToChildNode) + len(node.clusterIDs) + len(node.exactClusterIDByPattern)
}

func (d *Drain) Delete(cluster *LogCluster) {
	d.removeExactCluster(cluster.id)
	d.idToCluster.Remove(cluster.id)
}

func (d *Drain) findMatchingClusterForTokens(content string, tokens []string) *LogCluster {
	tokenCount := len(tokens)

	// at first level, children are grouped by token (word) count
	curNode, ok := d.rootNode.keyToChildNode[strconv.Itoa(tokenCount)]
	if !ok {
		// no template with same token count yet
		return nil
	}

	// we always end the token list with an <END> token, so <2 tokens is a
	// special case for an empty input string. In this case, we return the
	// single cluster in that group.
	if tokenCount < 2 {
		return d.idToCluster.Get(curNode.clusterIDs[0])
	}

	// otherwise, we need to find the leaf node for this log.
	curNodeDepth := 1
	for _, token := range tokens {
		// at max depth
		if curNodeDepth >= d.maxNodeDepth {
			break
		}

		// this is last token
		if curNodeDepth == tokenCount {
			break
		}

		keyToChildNode := curNode.keyToChildNode
		curNode, ok = keyToChildNode[d.config.ParamString]
		if !ok { // no wildcard node, try exact match
			curNode, ok = keyToChildNode[token]
		}
		if !ok { // no existing path
			return nil
		}
		curNodeDepth++
	}

	if cluster := d.findExactCluster(curNode, content); cluster != nil {
		return cluster
	}

	// get best match among all clusters with same prefix, or None if no match is above sim_th
	cluster := d.findBestClusterForTokens(curNode, tokens)
	if cluster != nil && curNode.exactClusterIDByPattern != nil {
		d.indexExactCluster(curNode, cluster, content)
	}
	return cluster
}

// findBestClusterForTokens finds the best match for a log message (represented
// as tokens) versus a list of clusters. A Match is considered better if it has
// a higher similarity or if it has the same similarity but more parameter tokens.
func (d *Drain) findBestClusterForTokens(node *Node, tokens []string) *LogCluster {
	var maxCluster *LogCluster

	clusterIDs := node.clusterIDs
	activeCount := 0
	maxSimilarity := -1.0
	maxParamCount := -1
	exactMatchFound := false
	for _, clusterID := range clusterIDs {
		// Try to retrieve cluster from cache. It may not exist due to eviction.
		// In that case, remove it while preserving the order of active clusters.
		// We do not update the access time here, because this may not be the
		// cluster we're looking for.
		cluster := d.idToCluster.GetQuietly(clusterID)
		if cluster == nil {
			continue
		}
		clusterIDs[activeCount] = clusterID
		activeCount++
		// An exact match cannot be improved, but finish the walk to compact
		// stale IDs.
		if exactMatchFound {
			continue
		}

		similarity, paramCount := cluster.tokenDistance(tokens)
		if paramCount < 0 {
			continue
		}
		if similarity > maxSimilarity || (similarity == maxSimilarity && paramCount > maxParamCount) {
			maxSimilarity = similarity
			maxParamCount = paramCount
			maxCluster = cluster
			exactMatchFound = similarity == 1
		}
	}
	node.clusterIDs = clusterIDs[:activeCount]
	if maxSimilarity < d.config.SimTh {
		return nil
	}
	return maxCluster
}

func (d *Drain) addClusterToRootNode(cluster *LogCluster) {
	tokenCount := len(cluster.Tokens)
	tokenCountStr := strconv.Itoa(tokenCount)

	curNode, ok := d.rootNode.keyToChildNode[tokenCountStr]
	if !ok {
		curNode = newNode()
		d.rootNode.keyToChildNode[tokenCountStr] = curNode
	}

	d.addClusterToNode(curNode, cluster, cluster.Tokens, tokenCount, 1)
}

func (d *Drain) findExactCluster(node *Node, content string) *LogCluster {
	if node.exactClusterIDByPattern == nil {
		return nil
	}
	clusterID, ok := node.exactClusterIDByPattern[content]
	if !ok {
		return nil
	}
	cluster := d.idToCluster.GetQuietly(clusterID)
	if cluster == nil {
		d.removeExactCluster(clusterID)
		return nil
	}
	if !cluster.isExact() {
		d.removeExactCluster(clusterID)
		return nil
	}
	return cluster
}

func (d *Drain) indexExactCluster(node *Node, cluster *LogCluster, content string) {
	if node.exactClusterIDByPattern == nil || !cluster.isExact() {
		return
	}

	// A scan may select an exact cluster for a different input that will
	// generalize during ingest. Do not index that transient exact pattern.
	// cluster.String returns cluster-owned memory, so the map never retains
	// the caller's content buffer.
	pattern := cluster.String()
	if content != "" && pattern != content {
		return
	}
	if currentID, ok := node.exactClusterIDByPattern[pattern]; ok {
		if currentID == cluster.id {
			return
		}
		current := d.idToCluster.GetQuietly(currentID)
		if current != nil && current.isExact() {
			return
		}
		d.removeExactCluster(currentID)
	}
	if current, ok := d.exactLocationByClusterID[cluster.id]; ok {
		if current.node == node && current.pattern == pattern {
			return
		}
		d.removeExactCluster(cluster.id)
	}
	if d.config.MaxClusters > 0 && len(d.exactLocationByClusterID) >= d.config.MaxClusters {
		return
	}

	node.exactClusterIDByPattern[pattern] = cluster.id
	d.exactLocationByClusterID[cluster.id] = exactLocation{node: node, pattern: pattern}
}

func (d *Drain) removeExactCluster(clusterID int) {
	location, ok := d.exactLocationByClusterID[clusterID]
	if !ok {
		return
	}
	if location.node.exactClusterIDByPattern[location.pattern] == clusterID {
		delete(location.node.exactClusterIDByPattern, location.pattern)
	}
	delete(d.exactLocationByClusterID, clusterID)
}

func (d *Drain) exactIndexCapacity() int {
	if d.config.MaxClusters > 0 {
		return min(exactMatchIndexThreshold, d.config.MaxClusters)
	}
	return exactMatchIndexThreshold
}

func (d *Drain) buildExactIndex(node *Node, added *LogCluster) {
	node.exactClusterIDByPattern = make(map[string]int, d.exactIndexCapacity())
	node.exactIndexBuilt = true
	// The added cluster is already in clusterIDs but is not cached until this
	// function returns.
	for _, clusterID := range node.clusterIDs[:len(node.clusterIDs)-1] {
		if cluster := d.idToCluster.GetQuietly(clusterID); cluster != nil {
			d.indexExactCluster(node, cluster, "")
		}
	}
	d.indexExactCluster(node, added, "")
}

func (d *Drain) addClusterToNode(curNode *Node, cluster *LogCluster, tokens []string, totalTokens int, currentDepth int) {
	// If we can't descend any further, add the cluster ID to the node.
	if currentDepth >= min(d.maxNodeDepth, totalTokens) {
		curNode.clusterIDs = append(curNode.clusterIDs, cluster.id)
		switch {
		case curNode.exactClusterIDByPattern != nil:
			d.indexExactCluster(curNode, cluster, "")
		case curNode.exactIndexBuilt:
			curNode.exactClusterIDByPattern = make(map[string]int, d.exactIndexCapacity())
			d.indexExactCluster(curNode, cluster, "")
		case len(curNode.clusterIDs) >= exactMatchIndexThreshold:
			d.buildExactIndex(curNode, cluster)
		}
		return
	}

	token := tokens[0]

	// If our heuristic says this is likely data, we use the param string as the
	// token instead. This is non-standard DRAIN, but it helps to identify
	// patterns more quickly.
	if d.tokenIsLikelyData(token) {
		token = d.config.ParamString
	}

	// If we've reached the max number of children, we collapse this node
	if len(curNode.keyToChildNode)+1 >= d.config.MaxChildren {
		token = d.config.ParamString
	}

	nextNode, ok := curNode.keyToChildNode[token]
	if !ok {
		nextNode = newNode()
		curNode.keyToChildNode[token] = nextNode
	}

	d.addClusterToNode(nextNode, cluster, tokens[1:], totalTokens, currentDepth+1)
}
