## GCS
In this example Tempo is configured to write data to GCS via [fake-gcs-server](https://github.com/fsouza/fake-gcs-server) which presents a GCS compatible API.

1. First start up the GCS stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
      Name                    Command               State                                 Ports                               
--------------------------------------------------------------------------------------------------------
gcs_gcs_1          /bin/fake-gcs-server -data ...   Up      0.0.0.0:4443->4443/tcp,:::4443->4443/tcp                          
gcs_grafana_1      /run.sh                          Up      0.0.0.0:3000->3000/tcp,:::3000->3000/tcp                          
gcs_k6-tracing_1   /k6-tracing run /example-s ...   Up                                                                        
gcs_prometheus_1   /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp,:::9090->9090/tcp                          
gcs_tempo_1        /tempo -config.file=/etc/t ...   Up      0.0.0.0:32791->14268/tcp,:::32791->14268/tcp,                     
                                                            0.0.0.0:3200->3200/tcp,:::3200->3200/tcp
```

2. If you're interested you can kind of see the wal/blocks as they are being created. Navigate to https://localhost:4443/storage/v1/b/tempo/o
to get a dump of all objects in the bucket.

3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces. Also notice that you can query Tempo metrics from the Prometheus data source setup in 
Grafana.

4. To stop the setup use -

```console
docker-compose down -v
```
