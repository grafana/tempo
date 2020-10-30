# Examples

These folders contain example deployments of Tempo.  They are a good resource for getting some basic configurations together.

#### Tempo Query

Note that even in the single binary deploys a second image named `tempo-query` is being deployed.  Tempo itself does not
provide a way to visualize traces and relies on [Jaeger Query](https://www.jaegertracing.io/docs/1.19/deployment/#query-service--ui) to do so.  `tempo-query` is [Jaeger Query](https://www.jaegertracing.io/docs/1.19/deployment/#query-service--ui) with a [GRPC Plugin](https://github.com/jaegertracing/jaeger/tree/master/plugin/storage/grpc)
that allows it to speak with Tempo.

Tempo Query is also the method by which Grafana queries traces.  Notice that the Grafana datasource in all examples is pointed to
`tempo-query`.

## Docker Compose

The [docker-compose](./docker-compose) examples are simpler and designed to show minimal configuration.  This is a great place
to get started with Tempo and learn about various trace discovery flows.

- [local storage](./docker-compose/readme.md#local-storage)
  - At its simplest Tempo only requires a few parameters that identify where to store traces.
- [s3/minio storage](./docker-compose/readme.md#s3)
  - To reduce complexity not all config options are exposed on the command line.  This example uses the minio/s3 backend with a config file.
- [Trace discovery with Loki](./docker-compose/readme.md#loki-derived-fields)
  - This example brings in Loki and shows how to use a log flow to discover traces.

## Jsonnet/Tanka

The [jsonnet](./tk) examples are more complex and show off the full range of configuration available to Tempo.  The
Helm and jsonnet examples are equivalent.  They are both provided for people who prefer different configuration
mechanisms.

- [single binary](./tk/readme.md#single-binary)
  - A single binary jsonnet deployment.  Valuable for getting started with advanced configuration.
- [microservices](./tk/readme.md#microservices)
  - Tempo as a set of independently scalable microservices.  This is recommended for high volume full production deployments.

## Helm

The [helm](./helm) examples are more complex and show off the full range of configuration available to Tempo.  The
Helm and jsonnet examples are equivalent.  They are both provided for people who prefer different configuration
mechanisms.

- [single binary](./helm/readme.md#single-binary)
  - A single binary jsonnet deployment.  Valuable for getting started with advanced configuration.
- [microservices](./helm/readme.md#microservices)
  - Tempo as a set of independently scalable microservices.  This is recommended for high volume full production deployments.