## Docker-compose

So you found your way to the docker compose examples?  This is a great place to
get started with Tempo and see some of the various configuration options

Refer to [getting-started](https://grafana.com/docs/tempo/latest/getting-started/docker-example/) for a walk-through using the single-binary example.

### Examples

The easiest example to start with is the [single-binary](./single-binary/). This example will run Tempo as a single binary, [xk6-client-tracing](https://github.com/grafana/xk6-client-tracing) 
to generate traces and Grafana to visualize trace data.

To use any example simply:

1. Navigate to the appropriate folder and run `docker-compose up`
1. Visit [Grafana Explore](http://localhost:3000/explore) and try some basic queries. See [the docs](https://grafana.com/docs/tempo/latest/traceql/construct-traceql-queries/) for help on more complex queries.  
  `{}` - basic search that finds everything  
  `{} | rate()` - rate of all spans
1. Visit [Traces Drilldown](http://localhost:3000/a/grafana-exploretraces-app/) for a queryless way to explore your data.
1. Connect your favorite LLM agent to our [MCP server](https://grafana.com/docs/tempo/latest/api_docs/mcp-server/)

### Features

See below for a list of all examples and the features they demonstrate

| Example | Deployment | Tenancy | Trace Ingestion | Storage | Other Features |
|---------|------------|---------|-----------------|---------|------------------|
| [Single Binary](./single-binary/) | Single binary | Single tenant | Alloy | S3 (MinIO) | vulture for data integrity, metrics generator, streaming queries, mcp |
| [Distributed](./distributed/) | Distributed microservices | Single tenant | Alloy | S3 (MinIO) | vulture for data integrity, metrics-generator, streaming queries, mcp |
| [Multitenant](./multitenant/) | Single binary | Multitenant | OTel Collector + Direct OTLP | Local filesystem | vulture for data integrity, multiple tenants (tenant-1, tenant-2), streaming queries, mcp |
| [Debug](./debug/) | Single binary | Single tenant | Direct OTLP | Local filesystem | vulture for data integrity, tempo-debug image for breakpoint debugging, streaming queries, mcp | 

### Build images (optional)

This step is not necessary, but it can be nice for local testing.  For any of the above examples rebuilding these
images will cause docker compose to use your local code when running the examples.

Run the following from the project root folder to build the `grafana/tempo:latest` image that is used in all the examples:

```console
make docker-images
```
