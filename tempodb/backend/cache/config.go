package cache

import "time"

type Config struct {
	Path           string        `yaml:"disk_path"`
	MaxDiskMBs     int           `yaml:"disk_max_mbs"`
	DiskPruneCount int           `yaml:"disk_prune_count"`
	DiskCleanRate  time.Duration `yaml:"disk_clean_rate"`
}
