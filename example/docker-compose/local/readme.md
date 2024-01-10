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
       Name                     Command               State                                   Ports                                 
-----------------------------------------------------------------------------------------------------------
local_grafana_1      /run.sh                          Up      0.0.0.0:3000->3000/tcp,:::3000->3000/tcp                              
local_k6-tracing_1   /k6-tracing run /example-s ...   Up                                                                            
local_prometheus_1   /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp,:::9090->9090/tcp                              
local_tempo_1        /tempo -config.file=/etc/t ...   Up      0.0.0.0:14268->14268/tcp,:::14268->14268/tcp,                         
                                                              0.0.0.0:3200->3200/tcp,:::3200->3200/tcp,                             
                                                              0.0.0.0:4317->4317/tcp,:::4317->4317/tcp,                             
                                                              0.0.0.0:4318->4318/tcp,:::4318->4318/tcp,                             
                                                              0.0.0.0:9411->9411/tcp,:::9411->9411/tcp 
```

2. If you're interested you can see the wal/blocks as they are being created.

```console
ls tempo-data/
```

3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces. Also notice that you can query Tempo metrics from the Prometheus data source setup in
Grafana.

4. Tail logs of a container (eg: tempo)
```bash
docker logs local_tempo_1 -f
```

5. To stop the setup use -

```console
docker-compose down -v
```

## search streaming over http

- need to set `traceQLStreaming` feature flag in Grafana
- need to enable `stream_over_http_enabled` in tempo by setting `stream_over_http_enabled: true` in the config file.

you can use Grafana or tempo-cli to make a query.

tempo-cli: `$ tempo-cli query api search "0.0.0.0:3200" --use-grpc "{}" "2023-12-05T08:11:18Z" "2023-12-05T08:12:18Z" --org-id="test"`
