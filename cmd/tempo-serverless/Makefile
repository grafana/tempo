.PHONY: build-docker build-zip test

# builds a docker image named "tempo-serverless" using the pack tool
#   https://github.com/buildpacks/pack
build-docker: 
	go mod vendor

	pack build \
	  --builder gcr.io/buildpacks/builder:v1 \
	  --env GOOGLE_RUNTIME=go \
	  --env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
	  --env GOOGLE_FUNCTION_TARGET=Handler \
	  tempo-serverless

	rm -rf vendor

# builds tempo-serverless.zip file that can be copied to GCS and deployed
# as a function
build-zip:
	go mod vendor

	rm -f tempo-serverless.zip
	zip tempo-serverless.zip ./* -r \
		-x Makefile                      # exclude Makefile

	rm -rf vendor

# builds the docker image and uses it for testing
test: build-docker
	go test