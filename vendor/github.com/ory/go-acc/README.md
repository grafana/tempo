# go-acc

A tool for reporting accurate Code Coverage in Golang. It is a cross platform (osx, windows, linux) adaption of the following bash script:

```bash
touch ./coverage.tmp
echo 'mode: atomic' > coverage.txt
go list ./... | grep -v /cmd | grep -v /vendor | xargs -n1 -I{} sh -c 'go test -race -covermode=atomic -coverprofile=coverage.tmp -coverpkg $(go list ./... | grep -v /vendor | tr "\n" ",") {} && tail -n +2 coverage.tmp >> coverage.txt || exit 255' && rm coverage.tmp
```

## Installation & Usage

```
$ go get github.com/ory/go-acc
$ go-acc
A tool for reporting accurate Code Coverage in Golang.

Usage:
  go-acc <packages...> [flags]

Examples:
$ go-acc github.com/some/package
$ go-acc -o my-coverfile.txt github.com/some/package
$ go-acc ./...
$ go-acc $(glide novendor)

Flags:
      --covermode string   Which code coverage mode to use (default "atomic")
      --ignore strings     Will ignore packages that contains any of these strings
  -o, --output string      Location for the output file (default "coverage.txt")
  -t, --toggle             Help message for toggle

```
