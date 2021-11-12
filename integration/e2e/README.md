To run the integration tests, use the following commands

```
# build latest image
make docker-tempo

# run all tests
go test -v ./integration/e2e/...

# run a particular test "TestMicroservices"
go test -v ./integration/e2e/... -run TestMicroservices$
```
