VERSION = $(shell godzil show-version)
COMMIT = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-s -w -X main.GitCommit=$(COMMIT)"
EXTERNAL_TOOLS = \
    golang.org/x/lint/golint            \
    github.com/Songmu/godzil/cmd/godzil \
    github.com/Songmu/ghch/cmd/ghch     \
    github.com/mattn/goveralls                \
    github.com/Songmu/gocredits/cmd/gocredits \
    golang.org/x/tools/cmd/cover

ifdef update
  u=-u
endif

export GO111MODULE=on

.PHONY: default
default: test

.PHONY: deps
deps:
	go get ${u} -d
	go mod tidy

# install external tools for this project
.PHONY: devel-deps
devel-deps: deps
	@for tool in $(EXTERNAL_TOOLS) ; do \
      echo "Installing $$tool" ; \
      GO111MODULE=off go get $$tool; \
    done

# build generate binary on './bin' directory.
.PHONY: build
build:
	go build -ldflags=$(BUILD_LDFLAGS) -o bin/ghr

.PHONY: bump
bump: devel-deps
	godzil release

CREDITS: go.sum devel-deps
	gocredits -w

.PHONY: crossbuild
crossbuild: CREDITS
	goxz -pv=v${VERSION} -build-ldflags=$(BUILD_LDFLAGS) \
        -arch=386,amd64 -d=./pkg/dist/v${VERSION}
	cd pkg/dist/v${VERSION} && shasum -a 256 * > ./v${VERSION}_SHASUMS

# install installs binary on $GOPATH/bin directory.
.PHONY: install
install:
	go install -ldflags=$(BUILD_LDFLAGS)

.PHONY: upload
upload: build devel-deps
	bin/ghr -v
	bin/ghr -body="$$(ghch --latest -F markdown)" v$(VERSION) pkg/dist/v$(VERSION)

.PHONY: test-all
test-all: lint test

.PHONY: test
test: deps
	go test -v -parallel=4 ./...

.PHONY: test-race
test-race:
	go test -v -race ./...

.PHONY: lint
lint: devel-deps
	go vet ./...
	golint -set_exit_status ./...

.PHONY: cover
cover:
	go test -coverprofile=cover.out
	go tool cover -html cover.out
	rm cover.out

.PHONY: release
release: bump crossbuild upload
