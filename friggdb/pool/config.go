package pool

type Config struct {
	maxWorkers int `yaml:"max_workers"`
	queueDepth int `yaml:"queue_depth"`
}
