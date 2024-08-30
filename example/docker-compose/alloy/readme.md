## Grafana Agent
This example highlights setting up the Grafana Alloy in a simple tracing pipeline.

1. Start up the stack.

```console
docker compose up -d
```

At this point, the following containers should be spun up:

```console
docker compose ps
```
```
NAME                 IMAGE                                       COMMAND                  SERVICE      CREATED          STATUS         PORTS
alloy-alloy-1        grafana/alloy:v1.3.1                        "/bin/alloy run /etc…"   alloy        48 seconds ago   Up 5 seconds   0.0.0.0:4319->4319/tcp, 0.0.0.0:12345->12345/tcp
alloy-grafana-1      grafana/grafana:11.2.0                      "/run.sh"                grafana      48 seconds ago   Up 5 seconds   0.0.0.0:3000->3000/tcp
alloy-k6-tracing-1   ghcr.io/grafana/xk6-client-tracing:v0.0.5   "/k6-tracing run /ex…"   k6-tracing   48 seconds ago   Up 4 seconds   
alloy-prometheus-1   prom/prometheus:latest                      "/bin/prometheus --c…"   prometheus   48 seconds ago   Up 5 seconds   0.0.0.0:9090->9090/tcp
alloy-tempo-1        grafana/tempo:latest                        "/tempo -config.file…"   tempo        48 seconds ago   Up 5 seconds   0.0.0.0:57508->3200/tcp, 0.0.0.0:57509->4317/tcp, 0.0.0.0:57505->4318/tcp, 0.0.0.0:57506->9411/tcp, 0.0.0.0:57507->14268/tcp
```

2. If you're interested you can see the wal/blocks as they are being created.
```console
ls tempo-data/
```

3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces.

4. To stop the setup use:

```console
docker compose down -v
```
