## OpenTelemetry Collector
This example highlights setting up the OpenTelemetry Collector in a simple tracing pipeline.

1. First start up the stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
             Name                            Command               State                          Ports                       
----------------------------------------------------------------------------------------------------------------------
otel-collector_grafana_1          /run.sh                          Up      0.0.0.0:3000->3000/tcp,:::3000->3000/tcp           
otel-collector_k6-tracing_1       /k6-tracing run /example-s ...   Up                                                         
otel-collector_otel-collector_1   /otelcol --config=/etc/ote ...   Up      4317/tcp, 55678/tcp, 55679/tcp                     
otel-collector_prometheus_1       /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp,:::9090->9090/tcp           
otel-collector_tempo_1            /tempo -config.file=/etc/t ...   Up      0.0.0.0:32792->14268/tcp,:::32792->14268/tcp,      
                                                                           0.0.0.0:32796->3200/tcp,:::32796->3200/tcp,        
                                                                           0.0.0.0:32795->4317/tcp,:::32795->4317/tcp,        
                                                                           0.0.0.0:32794->4318/tcp,:::32794->4318/tcp,        
                                                                           0.0.0.0:32793->9411/tcp,:::32793->9411/tcp 
```

2. If you're interested you can see the wal/blocks as they are being created.

```console
ls tempo-data/
```

3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces. Also notice that you can query Tempo metrics from the Prometheus data source setup in
Grafana.

4. To stop the setup use -

```console
docker-compose down -v
```
