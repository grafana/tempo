package debug

import (
	"encoding/hex"
	"fmt"
	"io"
)

func ReaderAt(reader io.ReaderAt, prefix string) io.ReaderAt {
	return &ioReaderAt{
		reader: reader,
		prefix: prefix,
	}
}

type ioReaderAt struct {
	reader io.ReaderAt
	prefix string
}

func (d *ioReaderAt) ReadAt(b []byte, off int64) (int, error) {
	n, err := d.reader.ReadAt(b, off)
	fmt.Printf("%s: Read(%d) @%d => %d %v \n%s\n", d.prefix, len(b), off, n, err, hex.Dump(b[:n]))
	return n, err
}

func Reader(reader io.Reader, prefix string) io.Reader {
	return &ioReader{
		reader: reader,
		prefix: prefix,
	}
}

type ioReader struct {
	reader io.Reader
	prefix string
	offset int64
}

func (d *ioReader) Read(b []byte) (int, error) {
	n, err := d.reader.Read(b)
	fmt.Printf("%s: Read(%d) @%d => %d %v \n%s\n", d.prefix, len(b), d.offset, n, err, hex.Dump(b[:n]))
	d.offset += int64(n)
	return n, err
}

func Writer(writer io.Writer, prefix string) io.Writer {
	return &ioWriter{
		writer: writer,
		prefix: prefix,
	}
}

type ioWriter struct {
	writer io.Writer
	prefix string
	offset int64
}

func (d *ioWriter) Write(b []byte) (int, error) {
	n, err := d.writer.Write(b)
	fmt.Printf("%s: Write(%d) @%d => %d %v \n  %q\n", d.prefix, len(b), d.offset, n, err, b[:n])
	d.offset += int64(n)
	return n, err
}
