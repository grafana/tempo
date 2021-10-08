## Scalable Single Binary

In this example Tempo is configured to write data to MinIO which presents an
S3-compatible API.  Additionally, `memberlist` is enabled to demonstrate how a
single binary can run all services and still make use of the cluster-awareness
that `memberlist` provides.

1. First start up the local stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
NAME                                                COMMAND                  SERVICE                    STATUS              PORTS
scalable-single-binary-grafana-1                    "/run.sh"                grafana                    running             3000/tcp
scalable-single-binary-minio-1                      "sh -euc 'mkdir -p /…"   minio                      running             9000/tcp
scalable-single-binary-prometheus-1                 "/bin/prometheus --c…"   prometheus                 running             9090/tcp
scalable-single-binary-synthetic-load-generator-1   "./start.sh"             synthetic-load-generator   running             
scalable-single-binary-tempo1-1                     "/tempo -target=scal…"   tempo1                     running             
scalable-single-binary-tempo2-1                     "/tempo -target=scal…"   tempo2                     running             
scalable-single-binary-tempo3-1                     "/tempo -target=scal…"   tempo3                     running             
scalable-single-binary-vulture-1                    "/tempo-vulture -pro…"   vulture                    running
```

2. If you're interested you can see the WAL/blocks as they are being created.  Navigate to MinIO at
http://localhost:9001 and use the username/password of `tempo`/`supersecret`.

3. The synthetic-load-generator is now printing out trace IDs it's flushing into Tempo.  To view its logs use -

```console
docker-compose logs -f synthetic-load-generator
```
```
synthetic-load-generator_1  | 20/10/24 08:27:09 INFO ScheduledTraceGenerator: Emitted traceId 57aedb829f352625 for service frontend route /product
synthetic-load-generator_1  | 20/10/24 08:27:09 INFO ScheduledTraceGenerator: Emitted traceId 25fa96b9da24b23f for service frontend route /cart
synthetic-load-generator_1  | 20/10/24 08:27:09 INFO ScheduledTraceGenerator: Emitted traceId 15b3ad814b77b779 for service frontend route /shipping
synthetic-load-generator_1  | 20/10/24 08:27:09 INFO ScheduledTraceGenerator: Emitted traceId 3803db7d7d848a1a for service frontend route /checkout
```

Logs are in the form

```
Emitted traceId <traceid> for service frontend route /cart
```

Copy one of these trace IDs.

4. Navigate to [Grafana](http://localhost:3000/explore) and paste the trace ID to request it from Tempo.
Also notice that you can query Tempo metrics from the Prometheus data source setup in Grafana.

5. To stop the setup use:

```console
docker-compose down -v
```
