package azure

type Config struct {
	StorageAccountName string `yaml:"storage-account-name"`
	StorageAccountKey  string `yaml:"storage-account-key"`
	ContainerName      string `yaml:"container-name"`
	Endpoint           string `yaml:"endpoint-suffix"`
	MaxBuffers         int    `yaml:"max-buffers"`
	BufferSize         int    `yaml:"buffer-size"`
}
