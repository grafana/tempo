package drain

type LogCluster struct {
	id          int
	Size        int
	Tokens      []string
	Stringer    func([]string) string
	ParamString string
	cache       string
}

func (c *LogCluster) ingestTokens(tokens []string) {
	if len(tokens) != len(c.Tokens) {
		panic("attempt to create template from sequences with different token lengths")
	}
	for i := range tokens {
		if tokens[i] != c.Tokens[i] && c.Tokens[i] != c.ParamString {
			c.Tokens[i] = c.ParamString
			c.cache = ""
		}
	}
	c.Size++
}

func (c *LogCluster) tokenDistance(tokens []string) (float64, int) {
	if len(c.Tokens) != len(tokens) {
		panic("attempt to compare sequences with different token lengths")
	}

	similarTokens := 0
	paramCount := 0
	for i := range c.Tokens {
		clusterToken := c.Tokens[i]
		inputToken := tokens[i]

		switch clusterToken {
		case c.ParamString:
			paramCount++
		case inputToken:
			similarTokens++
		}
	}
	retVal := float64(similarTokens) / float64(len(c.Tokens))
	return retVal, paramCount
}

func (c *LogCluster) String() string {
	if c.cache != "" {
		return c.cache
	}
	c.cache = c.Stringer(c.Tokens)
	return c.cache
}

func (c *LogCluster) GetTokens() []string {
	return c.Tokens
}
