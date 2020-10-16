---
title: Grafana Exemplars
draft: true
---

Integrating Tempo with Exemplars in Prometheus

Prometheus now supports an in-memory exemplar store that we can use to link traces
from data points in a prometheus histogram.

### Requirements

- Expose exemplars from the application

> Note: Exemplars can be exposed only from the Golang client application currently.



- Use the latest build of Prometheus to store metrics



- Use the latest build of Grafana to view Exemplars

- Configure Tempo Datasource in Grafana


### Sample setup for prototyping


- Clone a specific branch of the TNS repo
```
git clone --branch exemplars-env git@github.com:grafana/tns
```

- Enter TNS repo and build all tns docker images
```
cd tns && make build
```
- Start docker compose
```
cd production && docker-compose up -d
```
- At this point, the setup should look something like this:

```
Annanays-Mac:production annanay$ docker-compose ps
         Name                        Command               State                                  Ports
--------------------------------------------------------------------------------------------------------------------------------------
production_app_1          /app -log.level=debug http ...   Up      0.0.0.0:8001->80/tcp
production_db_1           /db -log.level=debug             Up      0.0.0.0:8000->80/tcp
production_jaeger_1       /go/bin/all-in-one-linux - ...   Up      14250/tcp, 14268/tcp, 0.0.0.0:8004->16686/tcp, 5775/udp, 5778/tcp, 6831/udp, 6832/udp
production_loadgen_1      /loadgen -log.level=debug  ...   Up      0.0.0.0:8002->80/tcp
production_prometheus_1   /bin/prometheus --config.f ...   Up      0.0.0.0:8003->9090/tcp
```

- The prometheus is already set up to scrape the tns containers. So the next step is to query prometheus using the new query API for exemplars (query here is on the metric `tns_request_duration_seconds_bucket` which is a histogram and exposes exemplars)

```console
$ curl -g 'http://localhost:8003/api/v1/query_exemplar?query=tns_request_duration_seconds_bucket'

{"status":"success","data":[{"seriesLabels":{"__name__":"tns_request_duration_seconds_bucket","instance":"app:80","job":"prometheus","le":"0.005","method":"GET","route":"metrics","status_code":"200","ws":"false"},"exemplars":[{"labels":{"traceID":"9f08bfc7a2bb2e"},"value":0.0017401,"timestamp":1600164378369,"hasTimestamp":true},
```

- Open http://localhost:8004 in the browser to see the Jaeger Search UI, and search for any trace from the queried exemplars. It should return a valid trace.
