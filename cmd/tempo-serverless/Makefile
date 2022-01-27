# todo: get docker run working?
# PACK=docker run -u ${shell id -u} -v /var/run/docker.sock:/var/run/docker.sock -v $(PWD):/workspace -w /workspace buildpacksio/pack
PACK=pack
GIT_REVISION := $(shell git rev-parse --short HEAD)

IN_CLOUD_FUNCTIONS=cd cloud-functions &&
IN_LAMBDA=cd lambda &&

#
# build docker images for local testing and code zip files for google cloud functions
#
.PHONY: build-gcf-docker
build-gcf-docker: 
	$(IN_CLOUD_FUNCTIONS) go mod vendor
	$(IN_CLOUD_FUNCTIONS) $(PACK) build tempo-serverless \
	  								--builder gcr.io/buildpacks/builder:v1 \
	  								--env GOOGLE_RUNTIME=go \
	  								--env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
	  								--env GOOGLE_FUNCTION_TARGET=Handler 	  
	$(IN_CLOUD_FUNCTIONS) rm -rf vendor

.PHONY: build-gcf-zip
build-gcf-zip:
	$(IN_CLOUD_FUNCTIONS) go mod vendor
	$(IN_CLOUD_FUNCTIONS) zip tempo-serverless-$(GIT_REVISION).zip ./* -r
	$(IN_CLOUD_FUNCTIONS) rm -rf vendor

#
# build docker images for local testing and code zip files for aws lambda
#
.PHONY: build-lambda-docker
build-lambda-docker:
#   todo: build exe in docker?
#	$(IN_LAMBDA) go mod vendor
	$(IN_LAMBDA) go build -o ./lambda
	$(IN_LAMBDA) docker build -t tempo-serverless .
	$(IN_LAMBDA) rm ./lambda
#	$(IN_LAMBDA) rm -rf vendor

.PHONY: build-lambda-zip
build-lambda-zip:
	$(IN_LAMBDA) go mod vendor
	$(IN_LAMBDA) zip tempo-serverless-$(GIT_REVISION).zip ./* -r \
		                   -x Dockerfile
	$(IN_LAMBDA) rm -rf vendor

.PHONY: test
test:
	go test -v .


# go build .
# docker run --rm -e DOCKER_LAMBDA_WATCH=1 -e DOCKER_LAMBDA_STAY_OPEN=1 -p 9001:9001 -v "$PWD":/var/task lambci/lambda:go1.x lambda