package unsupported

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/google/uuid"
)

func ownsWALBlock(entry fs.DirEntry) bool {
	// all new wal blocks are folders
	if !entry.IsDir() {
		return false
	}

	// We own anything that parses and contains "preview" in the version
	_, _, version, err := parseName(entry.Name())
	if err != nil {
		return false
	}

	return strings.Contains(version, "preview")
}

func parseName(filename string) (uuid.UUID, string, string, error) {
	splits := strings.Split(filename, "+")

	if len(splits) != 3 {
		return uuid.UUID{}, "", "", fmt.Errorf("unable to parse %s. unexpected number of segments", filename)
	}

	// first segment is blockID
	id, err := uuid.Parse(splits[0])
	if err != nil {
		return uuid.UUID{}, "", "", fmt.Errorf("unable to parse %s. error parsing uuid: %w", filename, err)
	}

	// second segment is tenant
	tenant := splits[1]
	if len(tenant) == 0 {
		return uuid.UUID{}, "", "", fmt.Errorf("unable to parse %s. 0 length tenant", filename)
	}

	// third segment is version
	version := splits[2]

	return id, tenant, version, nil
}
