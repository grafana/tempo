package sharedconfig

type DimensionMappings struct {
	Name        string   `yaml:"name"`
	SourceLabel []string `yaml:"source_labels"`
	Join        string   `yaml:"join"`
}
