# archives [![Go Reference](https://pkg.go.dev/badge/github.com/mholt/archives.svg)](https://pkg.go.dev/github.com/mholt/archives) [![Linux](https://github.com/mholt/archives/actions/workflows/ubuntu-latest.yml/badge.svg)](https://github.com/mholt/archives/actions/workflows/ubuntu-latest.yml) [![Mac](https://github.com/mholt/archives/actions/workflows/macos-latest.yml/badge.svg)](https://github.com/mholt/archives/actions/workflows/macos-latest.yml) [![Windows](https://github.com/mholt/archives/actions/workflows/windows-latest.yml/badge.svg)](https://github.com/mholt/archives/actions/workflows/windows-latest.yml)

Introducing **mholt/archives** - a cross-platform, multi-format Go library for working with archives and compression formats with a unified API and as virtual file systems compatible with [`io/fs`](https://pkg.go.dev/io/fs). 
<!--A powerful and flexible library enjoins an elegant CLI in this generic replacement for several platform-specific and format-specific archive utilities.-->

## Features

- Stream-oriented APIs
- Automatically identify archive and compression formats:
	- By file name
	- By stream peeking (headers)
- Traverse directories, archives, and other files uniformly as [`io/fs`](https://pkg.go.dev/io/fs) file systems:
	- [`FileFS`](https://pkg.go.dev/github.com/mholt/archives#FileFS)
	- [`DirFS`](https://pkg.go.dev/github.com/mholt/archives#DirFS)
	- [`ArchiveFS`](https://pkg.go.dev/github.com/mholt/archives#ArchiveFS)
- Seamlessly walk into archive files using [`DeepFS`](https://pkg.go.dev/github.com/mholt/archives#DeepFS)
- Compress and decompress files
- Create and extract archive files
- Walk or traverse into archive files
- Extract only specific files from archives
- Insert into (append to) .tar and .zip archives without re-creating entire archive
- Numerous archive and compression formats supported
- Read from password-protected 7-Zip and RAR files
- Extensible (add more formats just by registering them)
- Cross-platform, static binary
- Pure Go (no cgo)
- Multithreaded Gzip
- Adjustable compression levels
- Super-fast Snappy implementation (via [S2](https://github.com/klauspost/compress/blob/master/s2/README.md))

### Supported compression formats

- brotli (.br)
- bzip2 (.bz2)
- flate (.zip)
- gzip (.gz)
- lz4 (.lz4)
- lzip (.lz)
- minlz (.mz)
- snappy (.sz) and S2 (.s2)
- xz (.xz)
- zlib (.zz)
- zstandard (.zst)

### Supported archive formats

- .zip
- .tar (including any compressed variants like .tar.gz)
- .rar (read-only)
- .7z (read-only)

## Command line utility

There is an independently-maintained command line tool called [**`arc`**](https://github.com/jm33-m0/arc) currently in development that will expose many of the functions of this library to a shell.

## Library use

```bash
$ go get github.com/mholt/archives
```


### Create archive

Creating archives can be done entirely without needing a real disk or storage device. All you need is a list of [`FileInfo` structs](https://pkg.go.dev/github.com/mholt/archives#FileInfo), which can be implemented without a real file system.

However, creating archives from a disk is very common, so you can use the [`FilesFromDisk()` function](https://pkg.go.dev/github.com/mholt/archives#FilesFromDisk) to help you map filenames on disk to their paths in the archive.

In this example, we add 4 files and a directory (which includes its contents recursively) to a .tar.gz file:

```go
ctx := context.TODO()

// map files on disk to their paths in the archive using default settings (second arg)
files, err := archives.FilesFromDisk(ctx, nil, map[string]string{
	"/path/on/disk/file1.txt": "file1.txt",
	"/path/on/disk/file2.txt": "subfolder/file2.txt",
	"/path/on/disk/file3.txt": "",              // put in root of archive as file3.txt
	"/path/on/disk/file4.txt": "subfolder/",    // put in subfolder as file4.txt
	"/path/on/disk/folder":    "Custom Folder", // contents added recursively
})
if err != nil {
	return err
}

// create the output file we'll write to
out, err := os.Create("example.tar.gz")
if err != nil {
	return err
}
defer out.Close()

// we can use the CompressedArchive type to gzip a tarball
// (since we're writing, we only set Archival, but if you're
// going to read, set Extraction)
format := archives.CompressedArchive{
	Compression: archives.Gz{},
	Archival:    archives.Tar{},
}

// create the archive
err = format.Archive(ctx, out, files)
if err != nil {
	return err
}
```

### Extract archive

Extracting an archive, extracting _from_ an archive, and walking an archive are all the same function.

Simply use your format type (e.g. `Zip`) to call `Extract()`. You'll pass in a context (for cancellation), the input stream, and a callback function to handle each file.

```go
// the type that will be used to read the input stream
var format archives.Zip

err := format.Extract(ctx, input, func(ctx context.Context, f archives.FileInfo) error {
	// do something with the file here; or, if you only want a specific file or directory,
	// just return until you come across the desired f.NameInArchive value(s)
	return nil
})
if err != nil {
	return err
}
```

### Identifying formats

When you have an input stream with unknown contents, this package can identify it for you. It will try matching based on filename and/or the header (which peeks at the stream):

```go
// unless your stream is an io.Seeker, use the returned stream value to
// ensure you re-read the bytes consumed during Identify()
format, stream, err := archives.Identify(ctx, "filename.tar.zst", stream)
if err != nil {
	return err
}

// you can now type-assert format to whatever you need

// want to extract something?
if ex, ok := format.(archives.Extractor); ok {
	// ... proceed to extract
}

// or maybe it's compressed and you want to decompress it?
if decomp, ok := format.(archives.Decompressor); ok {
	rc, err := decomp.OpenReader(unknownFile)
	if err != nil {
		return err
	}
	defer rc.Close()

	// read from rc to get decompressed data
}
```

`Identify()` works by reading an arbitrary number of bytes from the beginning of the stream (just enough to check for file headers). It buffers them and returns a new reader that lets you re-read them anew. If your input stream is `io.Seeker` however, no buffer is created as it uses `Seek()` instead, and the returned stream is the same as the input.

### Virtual file systems

This is my favorite feature.

Let's say you have a directory on disk, an archive, a compressed archive, any other regular file, or a stream of any of the above! You don't really care; you just want to use it uniformly no matter what it is.

Simply create a file system:

```go
// filename could be:
// - a folder ("/home/you/Desktop")
// - an archive ("example.zip")
// - a compressed archive ("example.tar.gz")
// - a regular file ("example.txt")
// - a compressed regular file ("example.txt.gz")
// and/or the last argument could be a stream of any of the above
fsys, err := archives.FileSystem(ctx, filename, nil)
if err != nil {
	return err
}
```

This is a fully-featured `fs.FS`, so you can open files and read directories, no matter what kind of file the input was.

For example, to open a specific file:

```go
f, err := fsys.Open("file")
if err != nil {
	return err
}
defer f.Close()
```

If you opened a regular file or archive, you can read from it. If it's a compressed file, reads are automatically decompressed.

If you opened a directory (either real or in an archive), you can list its contents:

```go
if dir, ok := f.(fs.ReadDirFile); ok {
	// 0 gets all entries, but you can pass > 0 to paginate
	entries, err := dir.ReadDir(0)
	if err != nil {
		return err
	}
	for _, e := range entries {
		fmt.Println(e.Extension())
	}
}
```

Or get a directory listing this way:

```go
entries, err := fsys.ReadDir("Playlists")
if err != nil {
	return err
}
for _, e := range entries {
	fmt.Println(e.Extension())
}
```

Or maybe you want to walk all or part of the file system, but skip a folder named `.git`:

```go
err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if path == ".git" {
		return fs.SkipDir
	}
	fmt.Println("Walking:", path, "Dir?", d.IsDir())
	return nil
})
if err != nil {
	return err
}
```

The `archives` package lets you do it all.

**Important .tar note:** Tar files do not efficiently implement file system semantics due to their historical roots in sequential-access design for tapes. File systems inherently assume some index facilitating random access, but tar files need to be read from the beginning to access something at the end. This is especially slow when the archive is compressed. Optimizations have been implemented to amortize `ReadDir()` calls so that `fs.WalkDir()` only has to scan the archive once, but they use more memory. Open calls require another scan to find the file. It may be more efficient to use `Tar.Extract()` directly if file system semantics are not important to you.

#### Use with `http.FileServer`

It can be used with http.FileServer to browse archives and directories in a browser. However, due to how http.FileServer works, don't directly use http.FileServer with compressed files; instead wrap it like following:

```go
fileServer := http.FileServer(http.FS(archiveFS))
http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
	// disable range request
	writer.Header().Set("Accept-Ranges", "none")
	request.Header.Del("Range")
	
	// disable content-type sniffing
	ctype := mime.TypeByExtension(filepath.Ext(request.URL.Path))
	writer.Header()["Content-Type"] = nil
	if ctype != "" {
		writer.Header().Set("Content-Type", ctype)
	}
	fileServer.ServeHTTP(writer, request)
})
```

http.FileServer will try to sniff the Content-Type by default if it can't be inferred from file name. To do this, the http package will try to read from the file and then Seek back to file start, which the libray can't achieve currently. The same goes with Range requests. Seeking in archives is not currently supported by this package due to limitations in dependencies.

If Content-Type is desirable, you can [register it](https://pkg.go.dev/mime#AddExtensionType) yourself.

### Compress data

Compression formats let you open writers to compress data:

```go
// wrap underlying writer w
compressor, err := archives.Zstd{}.OpenWriter(w)
if err != nil {
	return err
}
defer compressor.Close()

// writes to compressor will be compressed
```

### Decompress data

Similarly, compression formats let you open readers to decompress data:

```go
// wrap underlying reader r
decompressor, err := archives.Snappy{}.OpenReader(r)
if err != nil {
	return err
}
defer decompressor.Close()

// reads from decompressor will be decompressed
```

### Append to tarball and zip archives

Tar and Zip archives can be appended to without creating a whole new archive by calling `Insert()` on a tar or zip stream. However, for tarballs, this requires that the tarball is not compressed (due to complexities with modifying compression dictionaries).

Here is an example that appends a file to a tarball on disk:

```go
tarball, err := os.OpenFile("example.tar", os.O_RDWR, 0644)
if err != nil {
	return err
}
defer tarball.Close()

// prepare a text file for the root of the archive
files, err := archives.FilesFromDisk(nil, map[string]string{
	"/home/you/lastminute.txt": "",
})

err := archives.Tar{}.Insert(context.Background(), tarball, files)
if err != nil {
	return err
}
```

The code is similar for inserting into a Zip archive, except you'll call `Insert()` on a `Zip{}` value instead.


### Traverse into archives while walking

If you are traversing/walking the file system using [`fs.WalkDir()`](https://pkg.go.dev/io/fs#WalkDir), the [**`DeepFS`**](https://pkg.go.dev/github.com/mholt/archives#DeepFS) type lets you walk the contents of archives (and compressed archives!) transparently as if the archive file was a regular directory on disk.

Simply root your DeepFS at a real path, then walk away:

```go
fsys := &archives.DeepFS{Root: "/some/dir"}

err := fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
	...
})
```

You'll notice that paths within archives look like `/some/dir/archive.zip/foo/bar.txt`. If you pass a path like that into `fsys.Open()`, it will split the path at the end of the archive file (`/some/dir/archive.zip`) and use the remainder of the path (`foo/bar.txt`) inside the archive.
