package azure

type Config struct {
	StorageAccountName string `yaml:"storage_account_name"`
	StorageAccountKey  string `yaml:"storage_account_key"`
	ContainerName      string `yaml:"container_name"`
	Endpoint           string `yaml:"endpoint_suffix"`
	MaxRetries         int    `yaml:"max_retries"`
	DevelopmentMode    bool   `yaml:"development_mode"`
}
