# Example

## Build images

Run the following from the project root folder to build `tempo:latest` and `tempo-query:latest` images
that will be used in the test environment:

```console
make docker-images
```

Note: `make docker-tempo` command needs to be run every time there are changes to the codebase.

## Docker Compose

```
cd docker-compose
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

## Jsonnet/Tanka

The Jsonnet is meant to be applied to with (tanka)[https://github.com/grafana/tanka].  To test the jsonnet locally requires:

- k3d > v1.6.0
- tanka > v0.8.0

```
cd tk
k3d create --name tempo \
           --publish 16686:80

export KUBECONFIG="$(k3d get-kubeconfig --name='tempo')"

# double check you're applying to your local k3d before running this!
#   deployment_type can be single-binary or microservices
tk apply <deployment_type>
```

After the applications are running check the load generators logs

```
kc logs synthetic-load-generator-75bfc5d545-xz5rz
...
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId e9f4add3ac7c7115 for service frontend route /product
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId 3890ea9c4d7fab00 for service frontend route /cart
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId c36fc5169bf0693d for service frontend route /cart
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId ebaf7d02b96b30fc for service frontend route /cart
20/03/03 21:30:02 INFO ScheduledTraceGenerator: Emitted traceId 23a09a0efd0d1ef0 for service frontend route /cart
```

Extract a trace id and view it in your browser at `http://localhost:16686/trace/<traceid>`

Clean up:
```
k3d delete --name tempo
```