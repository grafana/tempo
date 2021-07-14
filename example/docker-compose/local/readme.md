## Local Storage
In this example all data is stored locally in the `tempo-data` folder. Local storage is fine for experimenting with Tempo
or when using the single binary, but does not work in a distributed/microservices scenario.

1. First start up the local stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
                  Name                                 Command               State                         Ports
--------------------------------------------------------------------------------------------------------------------------------------
docker-compose_grafana_1                    /run.sh                          Up      0.0.0.0:3000->3000/tcp
docker-compose_prometheus_1                 /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp
docker-compose_synthetic-load-generator_1   ./start.sh                       Up
docker-compose_tempo_1                      /tempo -storage.trace.back ...   Up      0.0.0.0:32772->14268/tcp, 0.0.0.0:32773->3200/tcp
```

2. If you're interested you can see the wal/blocks as they are being created.

```console
ls tempo-data/
```

3. The synthetic-load-generator is now printing out trace ids it's flushing into Tempo.  To view its logs use -

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

Copy one of these trace ids.

4. Navigate to [Grafana](http://localhost:3000/explore) and paste the trace id to request it from Tempo.
Also notice that you can query Tempo metrics from the Prometheus data source setup in Grafana.

5. To stop the setup use -

```console
docker-compose down -v
```