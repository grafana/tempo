# tempo-servless

This folder is intended to wrap a simple handler for searching backend storage for traces. Eventually it
will encapsulate all of the code and tooling necessary to build and test this code as a serverless function 
in all major cloud providers.

This code has currently only been built and tested in GCS.

## Configuration
The serverless handler is setup to be configured via environment variables starting with the `TEMPO_` prefix.
```
TEMPO_GCS_BUCKET_NAME=some-random-gcs-bucket
TEMPO_GCS_CHUNK_BUFFER_SIZE=10485760
TEMPO_GCS_HEDGE_REQUESTS_AT=400ms
TEMPO_BACKEND=gcs
```
All fields in tempodb.Config are accessible using their all caps yaml names. Also config objects can be descended
using the `_` character. Note that in the above example `TEMPO_BCS_BUCKET_NAME` refers to tempodb.Config.GCS.BucketName.

## Make

### make build-docker

Builds a docker image named `tempo-serverlesss` that can be used easily for testing. This step uses [Buildpack](https://github.com/buildpacks/pack).
Initially it was thought that Buildpacks could be used to write platform agnostic handler code and build 
artifacts for different cloud providers, but unsure if this will pan out. For now we just use it as an easy
way to build a docker image that is then used in integration tests.

```
docker run --rm -p 8080:8080 tempo-serverless
curl http://localhost:8080
```

### make build-zip

This step builds an actual artifact that can be used in Google Cloud Functions. Upload the zip to a GCS bucket
and configure the function appropriately to use it.