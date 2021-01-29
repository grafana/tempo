package v1

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

// buffer up in memory and then write a big ol' compressed block o shit at once
//  used by CompleteBlock/CompactorBlock
// may need additional code?  i.e. a signal that it's "about to flush" triggering a compression

type bufferedAppender struct {
}

func NewBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) common.Appender {
	return &bufferedAppender{}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *bufferedAppender) Append(id common.ID, b []byte) error {
	return nil
}

func (a *bufferedAppender) Records() []*common.Record {
	return nil
}

func (a *bufferedAppender) Length() int {
	return 0
}

func (a *bufferedAppender) DataLength() uint64 {
	return 0
}

func (a *bufferedAppender) Complete() {

}
