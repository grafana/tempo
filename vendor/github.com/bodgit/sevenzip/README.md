[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/bodgit/sevenzip/badge)](https://securityscorecards.dev/viewer/?uri=github.com/bodgit/sevenzip)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/6882/badge)](https://www.bestpractices.dev/projects/6882)
[![GitHub release](https://img.shields.io/github/v/release/bodgit/sevenzip)](https://github.com/bodgit/sevenzip/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/bodgit/sevenzip/build.yml?branch=main)](https://github.com/bodgit/sevenzip/actions?query=workflow%3ABuild)
[![Coverage Status](https://coveralls.io/repos/github/bodgit/sevenzip/badge.svg?branch=master)](https://coveralls.io/github/bodgit/sevenzip?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/bodgit/sevenzip)](https://goreportcard.com/report/github.com/bodgit/sevenzip)
[![GoDoc](https://godoc.org/github.com/bodgit/sevenzip?status.svg)](https://godoc.org/github.com/bodgit/sevenzip)
![Go version](https://img.shields.io/badge/Go-1.22-brightgreen.svg)
![Go version](https://img.shields.io/badge/Go-1.21-brightgreen.svg)

# sevenzip

A reader for 7-zip archives inspired by `archive/zip`.

Current status:

* Pure Go, no external libraries or binaries needed.
* Handles uncompressed headers, (`7za a -mhc=off test.7z ...`).
* Handles compressed headers, (`7za a -mhc=on test.7z ...`).
* Handles password-protected versions of both of the above (`7za a -mhc=on|off -mhe=on -ppassword test.7z ...`).
* Handles archives split into multiple volumes, (`7za a -v100m test.7z ...`).
* Handles self-extracting archives, (`7za a -sfx archive.exe ...`).
* Validates CRC values as it parses the file.
* Supports ARM, BCJ, BCJ2, Brotli, Bzip2, Copy, Deflate, Delta, LZ4, LZMA, LZMA2, PPC, SPARC and Zstandard methods.
* Implements the `fs.FS` interface so you can treat an opened 7-zip archive like a filesystem.

More examples of 7-zip archives are needed to test all of the different combinations/algorithms possible.

## Frequently Asked Questions

### Why is my code running so slow?

Someone might write the following simple code:
```golang
func extractArchive(archive string) error {
        r, err := sevenzip.OpenReader(archive)
        if err != nil {
                return err
        }
        defer r.Close()

        for _, f := range r.File {
                rc, err := f.Open()
                if err != nil {
                        return err
                }
                defer rc.Close()

                // Extract the file
        }

        return nil
}
```
Unlike a zip archive where every file is individually compressed, 7-zip archives can have all of the files compressed together in one long compressed stream, supposedly to achieve a better compression ratio.
In a naive random access implementation, to read the first file you start at the beginning of the compressed stream and read out that files worth of bytes.
To read the second file you have to start at the beginning of the compressed stream again, read and discard the first files worth of bytes to get to the correct offset in the stream, then read out the second files worth of bytes.
You can see that for an archive that contains hundreds of files, extraction can get progressively slower as you have to read and discard more and more data just to get to the right offset in the stream.

This package contains an optimisation that caches and reuses the underlying compressed stream reader so you don't have to keep starting from the beginning for each file, but it does require you to call `rc.Close()` before extracting the next file.
So write your code similar to this:
```golang
func extractFile(file *sevenzip.File) error {
        rc, err := f.Open()
        if err != nil {
                return err
        }
        defer rc.Close()

        // Extract the file

        return nil
}

func extractArchive(archive string) error {
        r, err := sevenzip.OpenReader(archive)
        if err != nil {
                return err
        }
        defer r.Close()

        for _, f := range r.File {
                if err = extractFile(f); err != nil {
                        return err
                }
        }

        return nil
}
```
You can see the main difference is to not defer all of the `Close()` calls until the end of `extractArchive()`.

There is a set of benchmarks in this package that demonstrates the performance boost that the optimisation provides, amongst other techniques:
```
$ go test -v -run='^$' -bench='Reader$' -benchtime=60s
goos: darwin
goarch: amd64
pkg: github.com/bodgit/sevenzip
cpu: Intel(R) Core(TM) i9-8950HK CPU @ 2.90GHz
BenchmarkNaiveReader
BenchmarkNaiveReader-12                  	       2	31077542628 ns/op
BenchmarkOptimisedReader
BenchmarkOptimisedReader-12              	     434	 164854747 ns/op
BenchmarkNaiveParallelReader
BenchmarkNaiveParallelReader-12          	     240	 361869339 ns/op
BenchmarkNaiveSingleParallelReader
BenchmarkNaiveSingleParallelReader-12    	     412	 171027895 ns/op
BenchmarkParallelReader
BenchmarkParallelReader-12               	     636	 112551812 ns/op
PASS
ok  	github.com/bodgit/sevenzip	472.251s
```
The archive used here is just the reference LZMA SDK archive, which is only 1 MiB in size but does contain 630+ files split across three compression streams.
The only difference between BenchmarkNaiveReader and the rest is the lack of a call to `rc.Close()` between files so the stream reuse optimisation doesn't take effect.

Don't try and blindly throw goroutines at the problem either as this can also undo the optimisation; a naive implementation that uses a pool of multiple goroutines to extract each file ends up being nearly 50% slower, even just using a pool of one goroutine can end up being less efficient.
The optimal way to employ goroutines is to make use of the `sevenzip.FileHeader.Stream` field; extract files with the same value using the same goroutine.
This achieves a 50% speed improvement with the LZMA SDK archive, but it very much depends on how many streams there are in the archive.

In general, don't try and extract the files in a different order compared to the natural order within the archive as that will also undo the optimisation.
The worst scenario would likely be to extract the archive in reverse order.

### Detecting the wrong password

It's virtually impossible to _reliably_ detect the wrong password versus some other corruption in a password protected archive.
This is partly due to how CBC decryption works; with the wrong password you don't get any sort of decryption error, you just a stream of bytes that aren't the correct ones.
This manifests itself when the file has been compressed _and_ encrypted; during extraction the file is decrypted and then decompressed so with the wrong password the decompression algorithm gets handed a stream which isn't valid so that's the error you see.

A `sevenzip.ReadError` error type can be returned for certain operations.
If `sevenzip.ReadError.Encrypted` is `true` then encryption is involved and you can use that as a **hint** to either set a password or try a different one.
Use `errors.As()` to check like this:
```golang
r, err := sevenzip.OpenReaderWithPassword(archive, password)
if err != nil {
        var e *sevenzip.ReadError
        if errors.As(err, &e) && e.Encrypted {
                // Encryption involved, retry with a different password
        }

        return err
}
```
Be aware that if the archive does not have the headers encrypted, (`7za a -mhe=off -ppassword test.7z ...`), then you can always open the archive and the password is only used when extracting the files.

If files are added to the archive encrypted and _not_ compressed, (`7za a -m0=copy -ppassword test.7z ...`), then you will never get an error extracting with the wrong password as the only consumer of the decrypted content will be your own code. To detect a potentially wrong password, calculate the CRC value and check that it matches the value in `sevenzip.FileHeader.CRC32`.
