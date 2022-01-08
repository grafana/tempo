package pool

type Config struct {
	MaxWorkers int `yaml:"max_workers"`
	QueueDepth int `yaml:"queue_depth"`
}

// default is concurrency disabled
func defaultConfig() *Config {
	return &Config{
		MaxWorkers: 30,
		QueueDepth: 10000,
	}
}
