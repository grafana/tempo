### Grafana 7.4.x
All of the other examples are designed to work with Grafana 7.5.x and forward. If you are using Grafana 7.4.x or before then you need
to use tempo-query along with Tempo for querying. This example shows all the configuration points necessary
to pull this off.

1. First start up the stack.

```console
docker-compose -f docker-compose.grafana7.4.yaml up -d
```

At this point, the following containers should be spun up -

```console
docker-compose -f docker-compose.grafana7.4.yaml ps
```
```
                  Name                                Command               State                         Ports                      
-------------------------------------------------------------------------------------------------------------------------------------
dockercompose_grafana_1                    /run.sh                          Up      0.0.0.0:3000->3000/tcp                           
dockercompose_prometheus_1                 /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp                           
dockercompose_synthetic-load-generator_1   ./start.sh                       Up                                                       
dockercompose_tempo-query_1                /go/bin/query-linux --grpc ...   Up      0.0.0.0:16686->16686/tcp                         
dockercompose_tempo_1                      /tempo -config.file=/etc/t ...   Up      0.0.0.0:59541->14268/tcp, 0.0.0.0:59542->3100/tcp
```

2. If you're interested you can see the wal/blocks as they are being created.
```console
ls ./example-data/tempo
```

3. The synthetic-load-generator is now printing out trace ids it's flushing into Tempo.  To view its logs use -

```console
docker-compose logs -f synthetic-load-generator
```
```
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 48367daf25266daa for service frontend route /currency
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 10e50d2aca58d5e7 for service frontend route /cart
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 51a4ac1638ee4c63 for service frontend route /shipping
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 1219370c6a796a6d for service frontend route /product
```

Logs are in the form

```
Emitted traceId <traceid> for service frontend route /cart
```

Copy one of these trace ids.

4. Navigate to [Grafana](http://localhost:3000/explore?orgId=1&left=%5B%22now-1h%22,%22now%22,%22Tempo-Query%20(Grafana%207.4)%22,%7B%7D%5D) and paste the trace id to request it from Tempo. Note that you are using the "Tempo-Query" datasource which is specially configured to look at tempo-query.

5. To stop the setup use -

```console
docker-compose -f docker-compose.grafana7.4.yaml down -v
```
