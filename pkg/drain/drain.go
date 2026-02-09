package drain

import (
	"slices"
	"strconv"
)

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

		rootNode:    newNode(),
		idToCluster: newLogClusterCache(config.StaleClusterAge, config.MaxClusters, metrics.PatternsEvictedTotal, metrics.PatternsExpiredTotal),
	}

	return d
}

// Node is a node in the prefix tree.
type Node struct {
	keyToChildNode map[string]*Node
	clusterIDs     []int
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
	d.tokenBuffer = d.tokenizer.Tokenize(content, d.tokenBuffer)
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

	return d.train(d.tokenBuffer)
}

func (d *Drain) newCluster(tokens []string) *LogCluster {
	d.clustersCounter++
	return &LogCluster{
		Tokens:      slices.Clone(tokens),
		id:          d.clustersCounter,
		Size:        1,
		Stringer:    d.tokenizer.Join,
		ParamString: d.config.ParamString,
	}
}

func (d *Drain) train(tokens []string) *LogCluster {
	cluster := d.findMatchingClusterForTokens(tokens)

	if cluster == nil {
		cluster = d.newCluster(tokens)
		d.addClusterToRootNode(cluster)
		d.metrics.PatternsDetectedTotal.Inc()
	} else {
		cluster.ingestTokens(tokens)
	}

	d.idToCluster.Put(cluster)
	return cluster
}

// Prune removes old branches from the tree. We rely on the cache eviction
// algorithm to remove clusters from the cache, then this method will remove
// references to them in the tree.
func (d *Drain) Prune() {
	d.pruneTree(d.rootNode)
}

// pruneTree removes old branches from a node and its children. Nodes are pruned
// once they reference no valid clusters in their clusterIDs list.
func (d *Drain) pruneTree(node *Node) int {
	for key, child := range node.keyToChildNode {
		if d.pruneTree(child) == 0 {
			delete(node.keyToChildNode, key)
		}
	}

	node.clusterIDs = slices.DeleteFunc(node.clusterIDs, d.idToCluster.NotExists)
	return len(node.keyToChildNode) + len(node.clusterIDs)
}

func (d *Drain) Delete(cluster *LogCluster) {
	d.idToCluster.Remove(cluster.id)
}

func (d *Drain) findMatchingClusterForTokens(tokens []string) *LogCluster {
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

	// get best match among all clusters with same prefix, or None if no match is above sim_th
	cluster := d.findBestClusterForTokens(curNode.clusterIDs, tokens)
	return cluster
}

// findBestClusterForTokens finds the best match for a log message (represented
// as tokens) versus a list of clusters. A Match is considered better if it has
// a higher similarity or if it has the same similarity but more parameter tokens.
func (d *Drain) findBestClusterForTokens(clusterIDs []int, tokens []string) *LogCluster {
	var maxCluster *LogCluster

	maxSimilarity := -1.0
	maxParamCount := -1
	for _, clusterID := range clusterIDs {
		// Try to retrieve cluster from cache. It may not exist due to eviction.
		// In that case, we skip it for now, and this id will eventually be
		// removed from the node by Prune. We do not update the access time
		// here, because this may not be the cluster we're looking for.
		cluster := d.idToCluster.GetQuietly(clusterID)
		if cluster == nil {
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
		}
	}
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

	d.addClusterToNode(curNode, cluster.id, cluster.Tokens, tokenCount, 1)
}

func (d *Drain) addClusterToNode(curNode *Node, clusterID int, tokens []string, totalTokens int, currentDepth int) {
	// If we can't descend any further, add the cluster ID to the node.
	if currentDepth >= min(d.maxNodeDepth, totalTokens) {
		curNode.clusterIDs = append(curNode.clusterIDs, clusterID)
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

	d.addClusterToNode(nextNode, clusterID, tokens[1:], totalTokens, currentDepth+1)
}
