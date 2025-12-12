package drain

import (
	"strings"
)

type LogCluster struct {
	id         int
	Size       int
	Tokens     []string
	TokenState interface{}
	Stringer   func([]string, interface{}) string
	cache      string
}

func (c *LogCluster) String() string {
	if c.cache != "" {
		return c.cache
	}
	if c.Stringer != nil {
		c.cache = c.Stringer(c.Tokens, c.TokenState)
		return c.cache
	}
	c.cache = strings.Join(c.Tokens, "")
	return c.cache
}

func (c *LogCluster) GetTokens() []string {
	return c.Tokens
}

func (c *LogCluster) append() {
	c.Size++
}
