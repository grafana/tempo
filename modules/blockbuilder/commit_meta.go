package blockbuilder

import "fmt"

const (
	kafkaCommitMetaV1 = 1
)

// marshallCommitMeta generates the commit metadata string.
// commitRecTs: timestamp of the record which was committed (and not the commit time).
func marshallCommitMeta(commitRecTs int64) string {
	return fmt.Sprintf("%d,%d", kafkaCommitMetaV1, commitRecTs)
}

// unmarshallCommitMeta parses the commit metadata string.
// commitRecTs: timestamp of the record which was committed (and not the commit time).
func unmarshallCommitMeta(s string) (commitRecTs int64, err error) {
	if s == "" {
		return
	}
	var (
		version int
		metaStr string
	)
	_, err = fmt.Sscanf(s, "%d,%s", &version, &metaStr)
	if err != nil {
		return 0, fmt.Errorf("invalid commit metadata format: parse meta version: %w", err)
	}

	if version != kafkaCommitMetaV1 {
		return 0, fmt.Errorf("unsupported commit meta version %d", version)
	}
	_, err = fmt.Sscanf(metaStr, "%d", &commitRecTs)
	if err != nil {
		return 0, fmt.Errorf("invalid commit metadata format: %w", err)
	}
	return
}
