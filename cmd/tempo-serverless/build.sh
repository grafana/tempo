set -e

go mod vendor

rm -f tempo-serverless.zip
zip tempo-serverless.zip ./* -r -x build.sh

rm -rf vendor