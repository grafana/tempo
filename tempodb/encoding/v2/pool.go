package v2

import (
	"fmt"
	"io"
	"sync"

	"github.com/golang/snappy"
	"github.com/grafana/tempo/v2/tempodb/backend"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/prometheus/prometheus/util/pool"
)

const maxEncoding = backend.EncS2

// WriterPool is a pool of io.Writer
// This is used by every chunk to avoid unnecessary allocations.
type WriterPool interface {
	GetWriter(io.Writer) (io.WriteCloser, error)
	PutWriter(io.WriteCloser)
	ResetWriter(dst io.Writer, resetWriter io.WriteCloser) (io.WriteCloser, error)
	Encoding() backend.Encoding
}

// ReaderPool similar to WriterPool but for reading chunks.
type ReaderPool interface {
	GetReader(io.Reader) (io.Reader, error)
	PutReader(io.Reader)
	ResetReader(src io.Reader, resetReader io.Reader) (io.Reader, error)
	Encoding() backend.Encoding
}

var (
	// Gzip is the gnu zip compression pool
	Gzip = GzipPool{level: gzip.DefaultCompression}
	// Lz4_64k is the l4z compression pool, with 64k buffer size
	Lz4_64k = LZ4Pool{bufferSize: 1 << 16}
	// Lz4_256k uses 256k buffer
	Lz4_256k = LZ4Pool{bufferSize: 1 << 18}
	// Lz4_1M uses 1M buffer
	Lz4_1M = LZ4Pool{bufferSize: 1 << 20}
	// Lz4_4M uses 4M buffer
	Lz4_4M = LZ4Pool{bufferSize: 1 << 22}
	// Snappy is the snappy compression pool
	Snappy SnappyPool
	// Noop is the no compression pool
	Noop NoopPool
	// Zstd Pool
	Zstd = ZstdPool{}
	// S2 Pool
	S2 = S2Pool{}

	// BytesBufferPool is a bytes buffer used for lines decompressed.
	// Buckets [0.5KB,1KB,2KB,4KB,8KB]
	BytesBufferPool = pool.New(1<<9, 1<<13, 2, func(size int) interface{} { return make([]byte, 0, size) })
)

func GetWriterPool(enc backend.Encoding) (WriterPool, error) {
	r, err := getReaderPool(enc)
	if err != nil {
		return nil, err
	}

	return r.(WriterPool), nil
}

func getReaderPool(enc backend.Encoding) (ReaderPool, error) {
	switch enc {
	case backend.EncNone:
		return &Noop, nil
	case backend.EncGZIP:
		return &Gzip, nil
	case backend.EncLZ4_64k:
		return &Lz4_64k, nil
	case backend.EncLZ4_256k:
		return &Lz4_256k, nil
	case backend.EncLZ4_1M:
		return &Lz4_1M, nil
	case backend.EncLZ4_4M:
		return &Lz4_4M, nil
	case backend.EncSnappy:
		return &Snappy, nil
	case backend.EncZstd:
		return &Zstd, nil
	case backend.EncS2:
		return &S2, nil
	default:
		return nil, fmt.Errorf("Unknown pool encoding %d", enc)
	}
}

// GzipPool is a gun zip compression pool
type GzipPool struct {
	readers sync.Pool
	writers sync.Pool
	level   int
}

// Encoding implements WriterPool and ReaderPool
func (pool *GzipPool) Encoding() backend.Encoding {
	return backend.EncGZIP
}

// GetReader gets or creates a new CompressionReader and reset it to read from src
func (pool *GzipPool) GetReader(src io.Reader) (io.Reader, error) {
	if r := pool.readers.Get(); r != nil {
		reader := r.(*gzip.Reader)
		err := reader.Reset(src)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}
	reader, err := gzip.NewReader(src)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// PutReader places back in the pool a CompressionReader
func (pool *GzipPool) PutReader(reader io.Reader) {
	pool.readers.Put(reader)
}

// ResetReader implements ReaderPool
func (pool *GzipPool) ResetReader(src io.Reader, resetReader io.Reader) (io.Reader, error) {
	reader := resetReader.(*gzip.Reader)
	err := reader.Reset(src)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// GetWriter gets or creates a new CompressionWriter and reset it to write to dst
func (pool *GzipPool) GetWriter(dst io.Writer) (io.WriteCloser, error) {
	if w := pool.writers.Get(); w != nil {
		writer := w.(*gzip.Writer)
		writer.Reset(dst)
		return writer, nil
	}

	level := pool.level
	if level == 0 {
		level = gzip.DefaultCompression
	}
	w, err := gzip.NewWriterLevel(dst, level)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// PutWriter places back in the pool a CompressionWriter
func (pool *GzipPool) PutWriter(writer io.WriteCloser) {
	pool.writers.Put(writer)
}

// ResetWriter implements WriterPool
func (pool *GzipPool) ResetWriter(dst io.Writer, resetWriter io.WriteCloser) (io.WriteCloser, error) {
	writer := resetWriter.(*gzip.Writer)
	writer.Reset(dst)

	return writer, nil
}

// LZ4Pool is an pool...of lz4s...
type LZ4Pool struct {
	readers    sync.Pool
	writers    sync.Pool
	bufferSize uint32 // available values: 1<<16 (64k), 1<<18 (256k), 1<<20 (1M), 1<<22 (4M). Defaults to 4MB, if not set.
}

// Encoding implements WriterPool and ReaderPool
func (pool *LZ4Pool) Encoding() backend.Encoding {
	switch pool.bufferSize {
	case 1 << 16:
		return backend.EncLZ4_64k
	case 1 << 18:
		return backend.EncLZ4_256k
	case 1 << 20:
		return backend.EncLZ4_1M
	case 1 << 22:
		return backend.EncLZ4_4M
	}

	return backend.EncNone
}

// GetReader gets or creates a new CompressionReader and reset it to read from src
func (pool *LZ4Pool) GetReader(src io.Reader) (io.Reader, error) {
	var r *lz4.Reader
	if pooled := pool.readers.Get(); pooled != nil {
		r = pooled.(*lz4.Reader)
		r.Reset(src)
	} else {
		r = lz4.NewReader(src)
	}
	return r, nil
}

// PutReader places back in the pool a CompressionReader
func (pool *LZ4Pool) PutReader(reader io.Reader) {
	pool.readers.Put(reader)
}

// ResetReader implements ReaderPool
func (pool *LZ4Pool) ResetReader(src io.Reader, resetReader io.Reader) (io.Reader, error) {
	reader := resetReader.(*lz4.Reader)
	reader.Reset(src)
	return reader, nil
}

// GetWriter gets or creates a new CompressionWriter and reset it to write to dst
func (pool *LZ4Pool) GetWriter(dst io.Writer) (io.WriteCloser, error) {
	var w *lz4.Writer
	if fromPool := pool.writers.Get(); fromPool != nil {
		w = fromPool.(*lz4.Writer)
		w.Reset(dst)
	} else {
		w = lz4.NewWriter(dst)
	}
	err := w.Apply(
		lz4.ChecksumOption(false),
		lz4.BlockSizeOption(lz4.BlockSize(pool.bufferSize)),
		lz4.CompressionLevelOption(lz4.Fast),
	)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// PutWriter places back in the pool a CompressionWriter
func (pool *LZ4Pool) PutWriter(writer io.WriteCloser) {
	pool.writers.Put(writer)
}

// ResetWriter implements WriterPool
func (pool *LZ4Pool) ResetWriter(dst io.Writer, resetWriter io.WriteCloser) (io.WriteCloser, error) {
	writer := resetWriter.(*lz4.Writer)
	writer.Reset(dst)
	return writer, nil
}

// SnappyPool is a really cool looking pool.  Dang that pool is _snappy_.
type SnappyPool struct {
	readers sync.Pool
	writers sync.Pool
}

// Encoding implements WriterPool and ReaderPool
func (pool *SnappyPool) Encoding() backend.Encoding {
	return backend.EncSnappy
}

// GetReader gets or creates a new CompressionReader and reset it to read from src
func (pool *SnappyPool) GetReader(src io.Reader) (io.Reader, error) {
	if r := pool.readers.Get(); r != nil {
		reader := r.(*snappy.Reader)
		reader.Reset(src)
		return reader, nil
	}
	return snappy.NewReader(src), nil
}

// PutReader places back in the pool a CompressionReader
func (pool *SnappyPool) PutReader(reader io.Reader) {
	pool.readers.Put(reader)
}

// ResetReader implements ReaderPool
func (pool *SnappyPool) ResetReader(src io.Reader, resetReader io.Reader) (io.Reader, error) {
	reader := resetReader.(*snappy.Reader)
	reader.Reset(src)
	return reader, nil
}

// GetWriter gets or creates a new CompressionWriter and reset it to write to dst
func (pool *SnappyPool) GetWriter(dst io.Writer) (io.WriteCloser, error) {
	if w := pool.writers.Get(); w != nil {
		writer := w.(*snappy.Writer)
		writer.Reset(dst)
		return writer, nil
	}
	return snappy.NewBufferedWriter(dst), nil
}

// PutWriter places back in the pool a CompressionWriter
func (pool *SnappyPool) PutWriter(writer io.WriteCloser) {
	_ = writer.(*snappy.Writer).Close()
	pool.writers.Put(writer)
}

// ResetWriter implements WriterPool
func (pool *SnappyPool) ResetWriter(dst io.Writer, resetWriter io.WriteCloser) (io.WriteCloser, error) {
	writer := resetWriter.(*snappy.Writer)
	writer.Reset(dst)
	return writer, nil
}

// NoopPool is for people who think compression is for the weak
type NoopPool struct{}

// Encoding implements WriterPool and ReaderPool
func (pool *NoopPool) Encoding() backend.Encoding {
	return backend.EncNone
}

// GetReader gets or creates a new CompressionReader and reset it to read from src
func (pool *NoopPool) GetReader(src io.Reader) (io.Reader, error) {
	return src, nil
}

// PutReader places back in the pool a CompressionReader
func (pool *NoopPool) PutReader(io.Reader) {}

// ResetReader implements ReaderPool
func (pool *NoopPool) ResetReader(src io.Reader, _ io.Reader) (io.Reader, error) {
	return src, nil
}

type noopCloser struct {
	io.Writer
}

func (noopCloser) Close() error { return nil }

// GetWriter gets or creates a new CompressionWriter and reset it to write to dst
func (pool *NoopPool) GetWriter(dst io.Writer) (io.WriteCloser, error) {
	return noopCloser{dst}, nil
}

// PutWriter places back in the pool a CompressionWriter
func (pool *NoopPool) PutWriter(io.WriteCloser) {}

// ResetWriter implements WriterPool
func (pool *NoopPool) ResetWriter(dst io.Writer, _ io.WriteCloser) (io.WriteCloser, error) {
	return noopCloser{dst}, nil
}

// ZstdPool is a zstd compression pool
type ZstdPool struct {
	// sync pool cannot be used with zstd b/c it requires an explicit close to be called to free resources
}

// Encoding implements WriterPool and ReaderPool
func (pool *ZstdPool) Encoding() backend.Encoding {
	return backend.EncZstd
}

// GetReader gets or creates a new CompressionReader and reset it to read from src
func (pool *ZstdPool) GetReader(src io.Reader) (io.Reader, error) {
	reader, err := zstd.NewReader(src)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// PutReader places back in the pool a CompressionReader
func (pool *ZstdPool) PutReader(reader io.Reader) {
	r := reader.(*zstd.Decoder)
	r.Close()
}

// ResetReader implements ReaderPool
func (pool *ZstdPool) ResetReader(src io.Reader, resetReader io.Reader) (io.Reader, error) {
	reader := resetReader.(*zstd.Decoder)
	err := reader.Reset(src)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// GetWriter gets or creates a new CompressionWriter and reset it to write to dst
func (pool *ZstdPool) GetWriter(dst io.Writer) (io.WriteCloser, error) {
	w, err := zstd.NewWriter(dst)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// PutWriter places back in the pool a CompressionWriter
func (pool *ZstdPool) PutWriter(writer io.WriteCloser) {
	w := writer.(*zstd.Encoder)
	w.Close()
}

// ResetWriter implements WriterPool
func (pool *ZstdPool) ResetWriter(dst io.Writer, resetWriter io.WriteCloser) (io.WriteCloser, error) {
	writer := resetWriter.(*zstd.Encoder)
	writer.Reset(dst)
	return writer, nil
}

// S2Pool is one s short of s3
type S2Pool struct {
	readers sync.Pool
	writers sync.Pool
}

// Encoding implements WriterPool and ReaderPool
func (pool *S2Pool) Encoding() backend.Encoding {
	return backend.EncS2
}

// GetReader gets or creates a new CompressionReader and reset it to read from src
func (pool *S2Pool) GetReader(src io.Reader) (io.Reader, error) {
	if r := pool.readers.Get(); r != nil {
		reader := r.(*s2.Reader)
		reader.Reset(src)
		return reader, nil
	}
	return s2.NewReader(src), nil
}

// PutReader places back in the pool a CompressionReader
func (pool *S2Pool) PutReader(reader io.Reader) {
	pool.readers.Put(reader)
}

// ResetReader implements ReaderPool
func (pool *S2Pool) ResetReader(src io.Reader, resetReader io.Reader) (io.Reader, error) {
	reader := resetReader.(*s2.Reader)
	reader.Reset(src)
	return reader, nil
}

// GetWriter gets or creates a new CompressionWriter and reset it to write to dst
func (pool *S2Pool) GetWriter(dst io.Writer) (io.WriteCloser, error) {
	if w := pool.writers.Get(); w != nil {
		writer := w.(*s2.Writer)
		writer.Reset(dst)
		return writer, nil
	}
	// todo: review options and tune for wal compression? i.e. tons of small writes
	// consider:
	//  s2.WriterConcurrency(1)     - disables concurrency, given that we write and immediately force flush with Close, this might be preferable
	//  s2.WriterBlockSize(10*1024) - default block size is 1MB which is much larger than a normal write
	return s2.NewWriter(dst), nil
}

// PutWriter places back in the pool a CompressionWriter
func (pool *S2Pool) PutWriter(writer io.WriteCloser) {
	_ = writer.(*s2.Writer).Close()
	pool.writers.Put(writer)
}

// ResetWriter implements WriterPool
func (pool *S2Pool) ResetWriter(dst io.Writer, resetWriter io.WriteCloser) (io.WriteCloser, error) {
	writer := resetWriter.(*s2.Writer)
	writer.Reset(dst)
	return writer, nil
}
