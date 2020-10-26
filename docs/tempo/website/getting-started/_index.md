---
title: Getting started
draft: true
weight: 100
---

Getting started with Tempo is easy.  For an application already instrumented for tracing, this guide can help quickly set it up with Tempo. If you're looking for a demo application to play around with Tempo, skip to the [examples](#examples-with-demo-app) section at the bottom of this page.

## Setting up Tempo

### Spin up Tempo backend

First, set up a docker network as shown -

```
docker network create docker-tempo
```

Next, download the [configuration file](https://github.com/grafana/tempo/blob/master/example/docker-compose/tempo-local.yaml) using the following command -

```
curl -o tempo.yaml https://raw.githubusercontent.com/grafana/tempo/master/example/docker-compose/tempo-local.yaml
```

The config file above configures Tempo to listen on default ports for a number of protocols.
List of protocols and their default ports:

|  Protocol    |   Port  |
|  ---         |   ---   |
|  OpenTelemetry  | 55680 |  # Grafana Agent uses this.
|  Jaeger - Thrift Compact | 6831 |  # Jaeger Golang client uses this when used with JAEGER_AGENT_HOST & JAEGER_AGENT_PORT
|  Jaeger - Thrift Binary |  6832  |
|  Jaeger - Thrift HTTP |  14268 |  # Jaeger Golang client uses this when used with JAEGER_ENDPOINT
|  Jaeger - GRPC |  14250  | # Jaeger Agent uses this.
|  Zipkin  | 9411 |

Choose the port corresponding to the protocol you wish to use to send traces to Tempo. For this example we have used Jaeger - Thrift
Compact format (port 6831).

```
docker run -d --rm -p 6831:6831/udp --name tempo -v $(PWD)/tempo-local.yaml:/etc/tempo-local.yaml \
    --network docker-tempo \
    grafana/tempo:latest \
    -config.file=/etc/tempo-local.yaml
```

### Spin up Tempo Query container

Download the [configuration file](https://github.com/grafana/tempo/blob/master/example/docker-compose/tempo-query.yaml) using the following command -

```
curl -o tempo-query.yaml https://raw.githubusercontent.com/grafana/tempo/master/example/docker-compose/tempo-query.yaml
```

Use this config file to fire up the Tempo Query container -

```
docker run -d --rm -p 16686:16686 -v $(PWD)/tempo-query.yaml:/etc/tempo-query.yaml \
    --network docker-tempo \
    grafana/tempo-query:latest \
    --grpc-storage-plugin.configuration-file=/etc/tempo-query.yaml
```

Make sure the UI is accessible at http://localhost:16686. If the UI looks similar to the Jaeger Query UI, that's because it is! Tempo Query uses the Jaeger Query framework together with a hashicorp go-grpc plugin to query the Tempo backend.

### Send traces from the application to Tempo

Depending on the client SDK used for instrumentation, the parameters to configure might be different. The following example shows configuration parameters for applications instrumented with the [Jaeger Golang Client](https://github.com/jaegertracing/jaeger-client-go).

Set the following environment variables for the application -

```
JAEGER_AGENT_HOST=localhost              # or 'tempo' if running the application with docker
JAEGER_AGENT_PORT=6831
```

For a complete list of SDKs, visit the [OpenTelemetry Registry](https://opentelemetry.io/registry/?s=sdk).

### Query for traces

You're all set to use Tempo! Make sure you're logging trace ids in your application logs, because Tempo can only retrieve a trace when queried with its ID.

View the logs of the application, copy a traceID and paste it in the Query UI at http://localhost:16686 ("Search by Trace ID" - In the navbar at the top).

<p align="center"><img src="tempo-query-ui.png" alt="Tempo Query UI"></p>

Happy tracing!


## Examples with demo app

If you don't have an application to instrument at the moment, fret not! A number of [examples](https://github.com/grafana/tempo/tree/master/example) have been provided which show off various deployment and [configuration](../configuration) options.

Some highlights:
- [Configuration](https://github.com/grafana/tempo/blob/master/example/docker-compose/tempo.yaml)
  - Shows S3/minio config
  - Shows how to start all receivers with their default endpoints
  - Shows most configuration options
- [Docker-compose](https://github.com/grafana/tempo/blob/master/example/docker-compose/docker-compose.local.yaml)
  - Shows an extremely basic configuration.  Just use a few cli options to tell Tempo where to put traces.
- [Microservices](https://github.com/grafana/tempo/tree/master/example/tk)
  - This jsonnet based example shows a complete microservice based deployment.


## Grafana Agent

The Grafana Agent is already set up to use Tempo.  Refer to the [configuration](https://github.com/grafana/agent/blob/master/docs/configuration-reference.md#tempo_config) and [example](https://github.com/grafana/agent/blob/master/example/docker-compose/agent/config/agent.yaml) for details.
