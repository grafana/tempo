set -e

go mod vendor

pack build \
  --builder gcr.io/buildpacks/builder:v1 \
  --env GOOGLE_RUNTIME=go \
  --env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
  --env GOOGLE_FUNCTION_TARGET=Handler \
  tempo-serverless

rm -rf vendor