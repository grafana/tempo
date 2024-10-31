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

# Lambda zips expect a compiled executable in the root. The filename "bootstrap" is important here.
# The new AWS Lambda runtime expects an executable with the name "bootstrap" to be provided, the "handler" configuration is ignored when using the new runtime.
# See https://aws.amazon.com/blogs/compute/migrating-aws-lambda-functions-from-the-go1-x-runtime-to-the-custom-runtime-on-amazon-linux-2/ for more info.
.PHONY: build-lambda-zip
build-lambda-zip:
	$(IN_LAMBDA) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap
	$(IN_LAMBDA) zip tempo-serverless-$(VERSION).zip bootstrap
	$(IN_LAMBDA) rm bootstrap

.PHONY: test
test:
	go test -v .

### Tidy dependencies for tempo-serverless module
.PHONY: update-mod
update-mod:
	$(IN_LAMBDA) go mod tidy -e
	$(IN_CLOUD_RUN) go mod tidy -e
