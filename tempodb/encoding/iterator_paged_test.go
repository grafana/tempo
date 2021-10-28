package encoding

import (
	"context"
	"io"
	"testing"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type mockDataReader struct{}

func (m *mockDataReader) Read(context.Context, []common.Record, [][]byte, []byte) ([][]byte, []byte, error) {

}
func (m *mockDataReader) Close() {

}
func (m *mockDataReader) NextPage([]byte) ([]byte, uint32, error) {

}

type mockObjectReaderWriter struct{}

func (m *mockObjectReaderWriter) MarshalObjectToWriter(id common.ID, b []byte, w io.Writer) (int, error) {

}
func (m *mockObjectReaderWriter) UnmarshalObjectFromReader(r io.Reader) (common.ID, []byte, error) {

}
func (m *mockObjectReaderWriter) UnmarshalAndAdvanceBuffer(buffer []byte) ([]byte, common.ID, []byte, error) {

}

type mockIndexReader struct{}

func (m *mockIndexReader) At(ctx context.Context, i int) (*common.Record, error) {

}
func (m *mockIndexReader) Find(ctx context.Context, id common.ID) (*common.Record, int, error) {

}

var _ common.IndexReader = (*mockIndexReader)(nil)
var _ common.ObjectReaderWriter = (*mockObjectReaderWriter)(nil)
var _ common.DataReader = (*mockDataReader)(nil)

// TestIteratorPaged tests the iterator paging functionality
func TestIteratorPaged(t *testing.T) {
}
