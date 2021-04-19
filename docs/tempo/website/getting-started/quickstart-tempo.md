---
title: Quickstart with Tempo
---

If you are running an application locally that is already instrumented with tracing,
this guide will help set up the components required to send & query traces in Tempo.

## Step 1: Spin up Tempo backend

First, set up a docker network as shown -

```
docker network create docker-tempo
```

Next, download the [configuration file](https://github.com/grafana/tempo/blob/main/example/docker-compose/etc/tempo-local.yaml) using the following command -

```
curl -o tempo-local.yaml https://raw.githubusercontent.com/grafana/tempo/main/example/docker-compose/etc/tempo-local.yaml
```

The config file above configures Tempo to listen on default ports for a number of protocols.
List of protocols and their default ports:

|  Protocol    |   Port  |
|  ---         |   ---   |
|  OpenTelemetry  | 4317 |
|  Jaeger - Thrift Compact | 6831 |  # Jaeger Golang client uses this when used with JAEGER_AGENT_HOST & JAEGER_AGENT_PORT
|  Jaeger - Thrift Binary |  6832  |
|  Jaeger - Thrift HTTP |  14268 |  # Jaeger Golang client uses this when used with JAEGER_ENDPOINT
|  Jaeger - GRPC |  14250  | # Jaeger Agent uses this.
|  Zipkin  | 9411 |

Choose the port corresponding to the protocol you wish to use to send traces to Tempo. For this example we have used Jaeger - Thrift
Compact format (port 6831).

```
docker run -d --rm -p 6831:6831/udp --name tempo -v $(pwd)/tempo-local.yaml:/etc/tempo-local.yaml \
    --network docker-tempo \
    grafana/tempo:latest \
    -config.file=/etc/tempo-local.yaml
```

## Step 2: Spin up Grafana for visualizing traces

```
docker run -d --rm -p 3000:3000 \
    --network docker-tempo \
    -e GF_AUTH_ANONYMOUS_ENABLED=true \
    -e GF_AUTH_ANONYMOUS_ORG_ROLE=Admin \
    -e GF_AUTH_DISABLE_LOGIN_FORM=true \
    grafana/grafana:7.5.2
```

Make sure the UI is accessible at http://localhost:3000.

Next, configure the tempo datasource in Grafana.

<p align="center"><img src="../tempo-ds.png" alt="Tempo Datasource"></p>


## Step 3: Send traces from the application to Tempo

Depending on the client SDK used for instrumentation, the parameters to configure might be different.
The following example shows configuration parameters for applications instrumented with the [Jaeger Golang Client](https://github.com/jaegertracing/jaeger-client-go).

Set the following environment variables for the application -

```
JAEGER_AGENT_HOST=localhost     # or 'tempo' if running the application with docker
JAEGER_AGENT_PORT=6831
```

For a complete list of SDKs, visit the [OpenTelemetry Registry](https://opentelemetry.io/registry/?s=sdk).
Some SDKs (for ex: .NET, Java) also have support for autoinstrumentation.

## Step 4: Query for traces

You're all set to use Tempo! Make sure you're logging trace ids in your application logs, because Tempo can only retrieve a trace when queried with its ID.

View the logs of the application, copy a traceID and paste it in the Grafana Tempo datasource at http://localhost:3100. Happy tracing!

<p align="center"><img src="../grafana-query.png" alt="Grafana Query UI"></p>
