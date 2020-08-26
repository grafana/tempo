VERSION = $(shell cd cmd/retry && gobump show -r)
ifdef update
  u=-u
endif

test: deps
	go test ./...

deps:
	go get ${u} -d -v -t ./...

devel-deps: deps
	go get ${u} github.com/golang/lint/golint \
        github.com/mattn/goveralls            \
        github.com/motemen/gobump/cmd/gobump  \
        github.com/Songmu/goxz/cmd/goxz       \
        github.com/Songmu/ghch/cmd/ghch       \
        github.com/tcnksm/ghr

lint: devel-deps
	go vet ./...
	golint -set_exit_status ./...

cover: devel-deps
	goveralls

bump:
	_tools/releng

crossbuild: devel-deps
	$(eval ver = $(shell cd cmd/retry && gobump show -r))
	goxz -pv=v$(ver) -d=./dist ./cmd/retry

upload:
	ghr v$(VERSION) dist/v$(VERSION)

release: bump crossbuild upload

.PHONY: test deps devel-deps lint cover bump crossbuild upload release
