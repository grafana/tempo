# Zipkin Receiver

| Status                   |                   |
| ------------------------ |-------------------|
| Stability                | [beta]            |
| Supported pipeline types | traces            |
| Distributions            | [core], [contrib] |

This receiver receives spans from [Zipkin](https://zipkin.io/) (V1 and V2).

## Getting Started

All that is required to enable the Zipkin receiver is to include it in the
receiver definitions.

```yaml
receivers:
  zipkin:
```

The following settings are configurable:

- `endpoint` (default = 0.0.0.0:9411): host:port to which the receiver is going
  to receive data. The valid syntax is described at
  https://github.com/grpc/grpc/blob/master/doc/naming.md.

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md) including CORS
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Queuing, retry and timeout settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md)

[beta]: https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[core]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol