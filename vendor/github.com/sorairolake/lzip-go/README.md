<!--
SPDX-FileCopyrightText: 2024 Shun Sakai

SPDX-License-Identifier: Apache-2.0 OR MIT
-->

# lzip-go

[![CI][ci-badge]][ci-url]
[![Go Reference][reference-badge]][reference-url]
![Go version][go-version-badge]

**lzip-go** is an implementation of the [lzip compressed format] written in
pure [Go].

This package supports reading and writing of lzip compressed streams.

## Usage

To install this package:

```sh
go get -u github.com/sorairolake/lzip-go
```

### Example

Please see [`example_test.go`].

### Documentation

See the [documentation][reference-url] for more details.

## Command-line utility

This package includes a simple command-line utility for reading and writing of
lzip format compressed files.

### Installation

#### From source

```sh
go install github.com/sorairolake/lzip-go/cmd/glzip@latest
```

#### From binaries

The [release page] contains pre-built binaries for Linux, macOS, Windows and
others.

#### How to build

To build the command-line utility:

```sh
just build-cmd
```

To build a man page:

```sh
just build-man
```

The man page is generated in `docs/man/man1`. Note that [Asciidoctor] is
required when building the man page.

### Usage

Please see [`glzip(1)`].

## Minimum Go version

This package requires the minimum version of Go 1.22.

## Changelog

Please see [CHANGELOG.adoc].

## Contributing

Please see [CONTRIBUTING.adoc].

## Acknowledgment

The API of this package is based on the [`compress/gzip`] package.

This package uses the [`github.com/ulikunitz/xz/lzma`] package to encode and
decode LZMA streams.

## License

Copyright &copy; 2024 Shun Sakai (see [AUTHORS.adoc])

This package is distributed under the terms of either the _Apache License 2.0_
or the _MIT License_.

This project is compliant with version 3.2 of the [_REUSE Specification_]. See
copyright notices of individual files for more details on copyright and
licensing information.

[ci-badge]: https://img.shields.io/github/actions/workflow/status/sorairolake/lzip-go/CI.yaml?branch=develop&style=for-the-badge&logo=github&label=CI
[ci-url]: https://github.com/sorairolake/lzip-go/actions?query=branch%3Adevelop+workflow%3ACI++
[reference-badge]: https://img.shields.io/badge/Go-Reference-steelblue?style=for-the-badge&logo=go
[reference-url]: https://pkg.go.dev/github.com/sorairolake/lzip-go
[go-version-badge]: https://img.shields.io/github/go-mod/go-version/sorairolake/lzip-go?style=for-the-badge&logo=go
[lzip compressed format]: https://www.nongnu.org/lzip/manual/lzip_manual.html#File-format
[Go]: https://go.dev/
[`example_test.go`]: example_test.go
[release page]: https://github.com/sorairolake/lzip-go/releases
[Asciidoctor]: https://asciidoctor.org/
[`glzip(1)`]: docs/man/man1/glzip.1.adoc
[CHANGELOG.adoc]: CHANGELOG.adoc
[CONTRIBUTING.adoc]: CONTRIBUTING.adoc
[`compress/gzip`]: https://pkg.go.dev/compress/gzip
[`github.com/ulikunitz/xz/lzma`]: https://pkg.go.dev/github.com/ulikunitz/xz/lzma
[AUTHORS.adoc]: AUTHORS.adoc
[_REUSE Specification_]: https://reuse.software/spec/
