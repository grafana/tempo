set -e

go mod vendor

pack build test --builder=gcr.io/buildpacks/builder:v1

rm -rf vendor