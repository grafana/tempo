To run the integration tests, use the following commands


`-count=1` is passed to disable cache during test runs.

```
# build latest image
make docker-tempo
make docker-tempo-query

# run all tests
go test -count=1 -v ./integration/e2e/...

# run a particular test "TestMicroservices"
go test -count=1 -v ./integration/e2e/... -run TestMicroservices$

# build and run a particular test "TestMicroservicesWithKVStores"
make docker-tempo && go test -count=1 -v ./integration/e2e/... -run TestMicroservicesWithKVStores$

# run a single e2e tests with timeout
go test -timeout 3m -count=1 -v ./integration/e2e/... -run ^TestMultiTenantSearch$

# follow and watch logs while tests are running (assuming e2e test container is named tempo_e2e-tempo)
docker logs $(docker container ls -f name=tempo_e2e-tempo -q) -f
```
