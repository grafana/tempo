## Distributed

In this example tempo is configured in a distributed manner, where modules are
running as separate components and configured to write data to S3 via MinIO,
which presents an S3 compatible API.

1. First start up the distributed stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
                 Name                               Command               State                         Ports                       
------------------------------------------------------------------------------------------------------------------------------------
distributed_compactor_1                  /tempo -target=compactor - ...   Up      0.0.0.0:49662->3200/tcp,:::49652->3200/tcp        
distributed_distributor_1                /tempo -target=distributor ...   Up      0.0.0.0:49659->3200/tcp,:::49649->3200/tcp        
distributed_grafana_1                    /run.sh                          Up      0.0.0.0:3000->3000/tcp,:::3000->3000/tcp          
distributed_ingester-0_1                 /tempo -target=ingester -c ...   Up      0.0.0.0:49663->3200/tcp,:::49653->3200/tcp        
distributed_ingester-1_1                 /tempo -target=ingester -c ...   Up      0.0.0.0:49660->3200/tcp,:::49650->3200/tcp        
distributed_ingester-2_1                 /tempo -target=ingester -c ...   Up      0.0.0.0:49665->3200/tcp,:::49655->3200/tcp        
distributed_minio_1                      sh -euc mkdir -p /data/tem ...   Up      9000/tcp, 0.0.0.0:9001->9001/tcp,:::9001->9001/tcp
distributed_prometheus_1                 /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp,:::9090->9090/tcp          
distributed_querier_1                    /tempo -target=querier -co ...   Up      0.0.0.0:49664->3200/tcp,:::49654->3200/tcp        
distributed_query-frontend_1             /tempo -target=query-front ...   Up      0.0.0.0:49661->3200/tcp,:::49651->3200/tcp        
distributed_synthetic-load-generator_1   ./start.sh                       Up                                                        
```

2. If you're interested you can see the wal/blocks as they are being created.  Navigate to minio at
http://localhost:9001 and use the username/password of `tempo`/`supersecret`.

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
