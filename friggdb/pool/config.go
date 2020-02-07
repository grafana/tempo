package pool

type Config struct {
	MaxWorkers int `yaml:"max_workers"`
	QueueDepth int `yaml:"queue_depth"`
}
