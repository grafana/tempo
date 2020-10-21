---
title: Getting started
draft: true
weight: 100
---

Getting started with Tempo is easy. A number of [examples](https://github.com/grafana/tempo/tree/master/example) have been provided which show off various deployment and [configuration](../configuration) options.

Some highlights:
- [Configuration](https://github.com/grafana/tempo/blob/master/example/docker-compose/tempo.yaml)
  - Shows S3/minio config
  - Shows how to configure start all receivers with their default endpoints
  - Shows most configuration options
- [Docker-compose](https://github.com/grafana/tempo/blob/master/example/docker-compose/docker-compose.local.yaml)
  - Shows an extremely basic configuration.  Just use a few cli options to tell Tempo where to put traces.
- [Microservices](https://github.com/grafana/tempo/tree/master/example/tk)
  - This jsonnet based example shows a complete microservice based deployment.


## Grafana Agent

The Grafana Agent is already set up to use Tempo.  Refer to the [configuration](https://github.com/grafana/agent/blob/master/docs/configuration-reference.md#tempo_config) and [example](https://github.com/grafana/agent/blob/master/example/docker-compose/agent/config/agent.yaml) for details.
