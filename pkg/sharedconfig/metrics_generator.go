package sharedconfig

type DimensionMappings struct {
	Name        string   `yaml:"name" json:"name"`
	SourceLabel []string `yaml:"source_labels" json:"source_labels"`
	Join        string   `yaml:"join" json:"join"`
}
