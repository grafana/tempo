## OpenTelemetry Collector Multitenant
This example highlights setting up the OpenTelemetry Collector in a multitenant tracing pipeline.

1. Start up the stack.

```console
docker compose up -d
```

At this point, the following containers should be spun up:

```console
docker compose ps
```
```
                   Name                                  Command               State                          Ports                        
-----------------------------------------------------------------------------------------------------------------------------------
otel-collector-multitenant_grafana_1          /run.sh                          Up      0.0.0.0:3000->3000/tcp,:::3000->3000/tcp            
otel-collector-multitenant_k6-tracing_1       /k6-tracing run /example-s ...   Up                                                          
otel-collector-multitenant_otel-collector_1   /otelcol --config=/etc/ote ...   Up      4317/tcp, 55678/tcp, 55679/tcp                      
otel-collector-multitenant_prometheus_1       /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp,:::9090->9090/tcp            
otel-collector-multitenant_tempo_1            /tempo -multitenancy.enabl ...   Up      0.0.0.0:32802->14268/tcp,:::32802->14268/tcp,       
                                                                                       0.0.0.0:32806->3200/tcp,:::32806->3200/tcp,         
                                                                                       0.0.0.0:32805->4317/tcp,:::32805->4317/tcp,         
                                                                                       0.0.0.0:32804->4318/tcp,:::32804->4318/tcp,         
                                                                                       0.0.0.0:32803->9411/tcp,:::32803->9411/tcp  
```

2. If you're interested you can see the wal/blocks as they are being created.
```console
ls tempo-data/
```

3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces. Also notice that you can query Tempo metrics from the Prometheus data source setup in
Grafana.

4. To stop the setup use:

```console
docker compose down -v
```
