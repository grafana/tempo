.PHONY: build-docker build-zip test

# todo: get docker run working?
# PACK=docker run -u ${shell id -u} -v /var/run/docker.sock:/var/run/docker.sock -v $(PWD):/workspace -w /workspace buildpacksio/pack
PACK=pack
GIT_REVISION := $(shell git rev-parse --short HEAD)

# builds a docker image named "tempo-serverless" using the pack tool
#   https://github.com/buildpacks/pack
build-docker: 
	go mod vendor

	$(PACK) build tempo-serverless \
	  --builder gcr.io/buildpacks/builder:v1 \
	  --env GOOGLE_RUNTIME=go \
	  --env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
	  --env GOOGLE_FUNCTION_TARGET=Handler 	  
	  
	rm -rf vendor

# builds tempo-serverless.zip file that can be copied to GCS and deployed
# as a function
build-zip:
	go mod vendor

	zip tempo-serverless-$(GIT_REVISION).zip ./* -r \
		-x Makefile                      # exclude Makefile

	rm -rf vendor

test:
	go test -v .


### Tidy dependencies for tempo-serverless module
.PHONY: update-mod
update-mod:
	go mod tidy -e