.PHONY: build-docker build-zip test

# todo: get docker run working?
# PACK=docker run -u ${shell id -u} -v /var/run/docker.sock:/var/run/docker.sock -v $(PWD):/workspace -w /workspace buildpacksio/pack
PACK=pack
GIT_REVISION := $(shell git rev-parse --short HEAD)

IN_CLOUD_FUNCTIONS=cd cloud-functions &&
IN_LAMBDA=cd lambda &&

# builds a docker image named "tempo-serverless" using the pack tool
#   https://github.com/buildpacks/pack
build-docker: 
	$(IN_CLOUD_FUNCTIONS) go mod vendor

	$(IN_CLOUD_FUNCTIONS) $(PACK) build tempo-serverless \
	  								--builder gcr.io/buildpacks/builder:v1 \
	  								--env GOOGLE_RUNTIME=go \
	  								--env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
	  								--env GOOGLE_FUNCTION_TARGET=Handler 	  
	  
	$(IN_CLOUD_FUNCTIONS) rm -rf vendor

# builds tempo-serverless.zip file that can be copied to GCS and deployed
# as a Google Cloud Function
build-gcf-zip:
	$(IN_CLOUD_FUNCTIONS) go mod vendor
	$(IN_CLOUD_FUNCTIONS) zip tempo-serverless-$(GIT_REVISION).zip ./* -r \
		                   -x Makefile                      # exclude Makefile

	$(IN_CLOUD_FUNCTIONS) rm -rf vendor

# builds tempo-serverless.zip file that can be copied to S3 and deployed
# as a Lambda
build-lambda-zip:
	$(IN_LAMBDA) go mod vendor
	$(IN_LAMBDA) zip tempo-serverless-$(GIT_REVISION).zip ./* -r \
		                   -x Makefile                      # exclude Makefile

	$(IN_LAMBDA) rm -rf vendor

test:
	go test -v .