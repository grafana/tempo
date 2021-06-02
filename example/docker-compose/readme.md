## Docker-compose

So you found your way to the docker-compose examples?  This is a great place to get started with Tempo, learn
some basic configuration and learn about various trace discovery flows.

If you are interested in more complex configuration we would recommend the [tanka/jsonnet examples](../tk/readme.md).

### Examples

- [Loki Derived Fields](./readme.loki.md)   
  Highlights use of Loki derived fields to jump directly from logs -> traces.
- [Grafana Agent](./readme.agent.md)  
  Simple example using the Grafana Agent as a tracing pipeline.
- [OTel Collector](./readme.otelcol.md)  
  Simple example using the OpenTelemetry Collector as a tracing pipeline.
- [Multitenant](./readme.multitenant.md)  
  Uses the OpenTelemetry Collector in an advanced multitenant configuration.
- [Grafana 7.4.x](./readme.grafana7.4.md)  
  Uses tempo-query to allow for querying from Grafana 7.4 and before.

These examples show off configuration of different storage backends.

- [Local Storage](./readme.local.md)  
- [S3/Minio](./readme.s3.md)
- [Azure/Azurite](./readme.azure.md)
- [GCS/Fake](./readme.gcs.md)

### Build Images (Optional)

This step is not necessary, but it can be nice for local testing.  For any of the above examples rebuilding these
images will cause docker-compose to use whatever local code you have in the examples.

Run the following from the project root folder to build `tempo:latest` and `tempo-query:latest` images
that will be used in the test environment:

```console
make docker-images
```