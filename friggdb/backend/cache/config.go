package cache

import "time"

type Config struct {
	Path           string        `yaml:"disk-path"`
	MaxDiskMBs     int           `yaml:"disk-max-mbs"`
	DiskPruneCount int           `yaml:"disk-prune-count"`
	DiskCleanRate  time.Duration `yaml:"disk-clean-rate"`
}
