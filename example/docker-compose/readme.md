## Docker-compose

So you found your way to the docker-compose examples?  This is a great place to get started with Tempo, learn
some basic configuration and learn about various trace discovery flows.

If you are interested in more complex configuration we would recommend the [tanka/jsonnet examples](../tk/readme.md).

### Examples

The easiest example to start with is [Local Storage](local/readme.md): this example will run Tempo as a single binary
together with the synthetic-load-generator, to generate traces, and Grafana, to query Tempo.  Data is stored locally on
disk. 

The following examples showcase specific features or integrations:

- [Grafana Agent](agent/readme.md)  
  Simple example using the Grafana Agent as a tracing pipeline.
- [OpenTelemetry Collector](otel-collector/readme.md)  
  Simple example using the OpenTelemetry Collector as a tracing pipeline.
- [OpenTelemetry Collector Multitenant](otel-collector-multitenant/readme.md)  
  Uses the OpenTelemetry Collector in an advanced multitenant configuration.

These examples show off configuration of different storage backends:

- [Local Storage](local/readme.md)  
- [S3/Minio](s3/readme.md)
- [Azure/Azurite](azure/readme.md)

### Build Images (Optional)

This step is not necessary, but it can be nice for local testing.  For any of the above examples rebuilding these
images will cause docker-compose to use your local code when running the examples.

Run the following from the project root folder to build the`grafana/tempo:latest` image that is used in all the examples:

```console
make docker-images
```