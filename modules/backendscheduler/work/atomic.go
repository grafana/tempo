package work

import (
	"os"
	"path/filepath"
)

// atomicWriteFile writes data to a file atomically using a temporary file and rename.
// This prevents possible data corruption in case of a crash or interruption during the write operation.
func atomicWriteFile(data []byte, targetPath, tempPrefix string) error {
	dir := filepath.Dir(targetPath)

	// Create unique temporary file
	tempFile, err := os.CreateTemp(dir, tempPrefix+".tmp.")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	// Write data to temporary file
	_, err = tempFile.Write(data)
	if err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return err
	}

	// Sync and close temporary file
	err = tempFile.Sync()
	if err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return err
	}

	err = tempFile.Close()
	if err != nil {
		os.Remove(tempPath)
		return err
	}

	// Atomically move temp file to final location
	return os.Rename(tempPath, targetPath)
}
