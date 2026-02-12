package drain

import "time"

type Config struct {
	LogClusterDepth int
	SimTh           float64
	MaxChildren     int
	MinTokens       int
	MaxTokens       int
	MaxClusters     int
	StaleClusterAge time.Duration
	ParamString     string
}

func DefaultConfig() *Config {
	return &Config{
		// At training, if at the depth of LogClusterDepth there is a cluster with
		// similarity coefficient greater that SimTh, then the log message is added
		// to that cluster. Otherwise, a new cluster is created.
		//
		// LogClusterDepth should be equal to the number of constant tokens from
		// the beginning of the message that likely determine the message contents.
		//
		//  > In this step, Drain traverses from a 1-st layer node, which
		//  > is searched in step 2, to a leaf node. This step is based on
		//  > the assumption that tokens in the beginning positions of a log
		//  > message are more likely to be constants. Specifically, Drain
		//  > selects the next internal node by the tokens in the beginning
		//  > positions of the log message
		LogClusterDepth: 30,
		// SimTh is basically a ratio of matching/total in the cluster.
		// Cluster tokens: "foo <*> bar fred"
		//       Log line: "foo bar baz qux"
		//                  *   *   *   x
		// Similarity of these sequences is 0.75 (the distance)
		// Both SimTh and LogClusterDepth impact branching factor: the greater
		// LogClusterDepth and SimTh, the less the chance that there will be
		// "similar" clusters, but the greater the footprint.
		SimTh:       0.3,
		MaxChildren: 1000,
		ParamString: `<_>`,
		MinTokens:   3, // const + number suffix + <END>
		MaxTokens:   80,
		MaxClusters: 100000,
	}
}
