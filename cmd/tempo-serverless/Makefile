PACK=pack
VERSION=$(shell ../../tools/image-tag | cut -d, -f 1)

IN_CLOUD_RUN=cd cloud-run &&
IN_LAMBDA=cd lambda &&

LOWER_VERSION = `echo $(VERSION) | tr A-Z a-z`

#
# build docker images for local testing and code zip files for google cloud run
#
.PHONY: build-docker
build-docker: build-docker-lambda-test build-docker-gcr

#
# google cloud run
#
.PHONY: build-docker-gcr-binary
build-docker-gcr-binary:
	$(IN_CLOUD_RUN) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main

.PHONY: build-docker-gcr
build-docker-gcr: build-docker-gcr-binary
	$(IN_CLOUD_RUN) docker build -f ./Dockerfile -t tempo-serverless:$(LOWER_VERSION) .
	$(IN_CLOUD_RUN) rm main
	docker tag tempo-serverless:$(LOWER_VERSION) tempo-serverless:latest

#
# aws lambda
#
.PHONY: build-docker-lambda-test
build-docker-lambda-test:
	$(IN_LAMBDA) CGO_ENABLED=0 go build -o ./lambda
	$(IN_LAMBDA) docker build -f ./Dockerfile -t tempo-serverless-lambda .

# lambda zips expect a compiled executable in root. the filename "main" is important
# as that should the handler config option in aws
.PHONY: build-lambda-zip
build-lambda-zip:
	$(IN_LAMBDA) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main
	$(IN_LAMBDA) zip tempo-serverless-$(VERSION).zip main
	$(IN_LAMBDA) rm main

.PHONY: test
test:
	go test -v .

### Tidy dependencies for tempo-serverless module
 # todo: remove -compat arg when moving to 1.18
.PHONY: update-mod
update-mod:
	$(IN_LAMBDA) go mod tidy -e -compat=1.17
	$(IN_CLOUD_RUN) go mod tidy -e -compat=1.17
