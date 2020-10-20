## Docker-compose

This step is not necessary, but it can be nice for local testing.  Rebuilding these images will cause 
docker-compose to use whatever local code you have in the examples.

Run the following from the project root folder to build `tempo:latest` and `tempo-query:latest` images
that will be used in the test environment:

```console
make docker-images
```

## Local Storage

```
docker-compose -f docker-compose.local.yaml up
```



```
docker-compose up
```
- Tempo
  - http://localhost:3100
- Tempo-Query
  - http://localhost:16686
- Grafana
  - http://localhost:3000
- Prometheus
  - http://localhost:9090

### Find Traces

The synthetic-load-generator is now printing out trace ids it's flushing into Tempo.  Logs are in the form

`Emitted traceId 27896d4ea9c8429d for service frontend route /cart`

Copy and paste the trace id into tempo-query to retrieve it from Tempo.

## S3

## Loki