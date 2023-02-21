package filesystem

type FileSystem struct{}

const name = "filesystem"

func (self *FileSystem) Name() string {
	return name
}

func (self *FileSystem) Collect() (result interface{}, err error) {
	result, err = getFileSystemInfo()
	return
}
