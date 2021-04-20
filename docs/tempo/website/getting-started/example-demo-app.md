---
title: Examples with demo app
---

# Examples with demo app

If you don't have an application to instrument at the moment, fret not! A number of [examples](https://github.com/grafana/tempo/tree/main/example) have been provided which show off various deployment and [configuration]({{< relref "../configuration" >}}) options.

Some highlights:
- [Configuration](https://github.com/grafana/tempo/blob/main/example/docker-compose/etc/tempo-s3-minio.yaml)
  - Shows S3/minio config
  - Shows how to start all receivers with their default endpoints
  - Shows most configuration options
- [Docker-compose](https://github.com/grafana/tempo/blob/main/example/docker-compose/docker-compose.yaml)
  - Shows an extremely basic configuration with local storage.
- [Microservices](https://github.com/grafana/tempo/tree/main/example/tk)
  - This jsonnet based example shows a complete microservice based deployment.

