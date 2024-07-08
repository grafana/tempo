**Running the integration tests**


`-count=1` is passed to disable cache during test runs.

```sh
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

**How to debug Tempo while running an integration test**

1. Build latest debug image
```sh
    make docker-tempo-debug
```
2. Use the function ``NewTempoAllInOneDebug`` in your test to spin a Tempo instance with debug capabilities
3. Set a breakpoint after ``require.NoError(t, s.StartAndWaitReady(tempo))`` and before the action you want debug
4. Get the port of Delve debugger inside the container
```sh
    docker ps --format '{{.Ports}}'  
    # 0.0.0.0:53467->2345
```
5. Run the debugger against that port as is specified [here](https://github.com/grafana/tempo/tree/main/example/docker-compose/debug)
 

