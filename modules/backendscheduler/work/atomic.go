package work

import (
	"os"
	"path/filepath"
)

// atomicWriteFile writes data to a file atomically using a temporary file and rename.
// This prevents possible data corruption in case of a crash or interruption during the write operation.
func atomicWriteFile(data []byte, targetFile string) error {
	var (
		dir       = filepath.Dir(targetFile)
		tmpPrefix = filepath.Base(targetFile)
	)

	// Create unique temporary file
	tmpFile, err := os.CreateTemp(dir, tmpPrefix+".tmp.")
	if err != nil {
		return err
	}
	tmpFilePath := tmpFile.Name()

	// Write data to temporary file
	_, err = tmpFile.Write(data)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFilePath)
		return err
	}

	// Sync and close temporary file
	err = tmpFile.Sync()
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFilePath)
		return err
	}

	err = tmpFile.Close()
	if err != nil {
		os.Remove(tmpFilePath)
		return err
	}

	// Atomically move temp file to final location
	return os.Rename(tmpFilePath, targetFile)
}
