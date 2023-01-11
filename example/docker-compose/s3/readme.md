## S3

In this example tempo is configured to write data to S3 via MinIO which presents an S3 compatible API.

1. First start up the S3 stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
     Name                    Command               State                                  Ports                               
------------------------------------------------------------------------------------------------------------------------------
s3_grafana_1      /run.sh                          Up      0.0.0.0:3000->3000/tcp,:::3000->3000/tcp                           
s3_k6-tracing_1   /k6-tracing run /example-s ...   Up                                                                         
s3_minio_1        sh -euc mkdir -p /data/tem ...   Up      9000/tcp, 0.0.0.0:9001->9001/tcp,:::9001->9001/tcp                 
s3_prometheus_1   /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp,:::9090->9090/tcp                           
s3_tempo_1        /tempo -config.file=/etc/t ...   Up      0.0.0.0:32807->14268/tcp,:::32807->14268/tcp,                      
                                                           0.0.0.0:3200->3200/tcp,:::3200->3200/tcp 
```

2. If you're interested you can see the wal/blocks as they are being created.  Navigate to minio at
http://localhost:9001 and use the username/password of `tempo`/`supersecret`.

3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces. Also notice that you can query Tempo metrics from the Prometheus data source setup in
Grafana.

4. To stop the setup use -

```console
docker-compose down -v
```
