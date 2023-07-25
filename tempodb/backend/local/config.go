package local

type Config struct {
	Path string `yaml:"path"`
}

func (c *Config) PathMatches(other *Config) bool {
	return c.Path == other.Path
}
