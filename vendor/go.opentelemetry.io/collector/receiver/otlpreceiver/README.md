# OTLP Receiver

| Status                   |                       |
| ------------------------ | --------------------- |
| Stability                | traces [stable]       |
|                          | metrics [stable]      |
|                          | logs [beta]           |
| Supported pipeline types | traces, metrics, logs |
| Distributions            | [core], [contrib]     |

Receives data via gRPC or HTTP using [OTLP](
https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)
format.

## Getting Started

All that is required to enable the OTLP receiver is to include it in the
receiver definitions. A protocol can be disabled by simply not specifying it in
the list of protocols.

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:
```

The following settings are configurable:

- `endpoint` (default = 0.0.0.0:4317 for grpc protocol, 0.0.0.0:4318 http protocol):
  host:port to which the receiver is going to receive data. The valid syntax is
  described at https://github.com/grpc/grpc/blob/master/doc/naming.md.

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md) including CORS
- [HTTP settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Auth settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configauth/README.md)

## Writing with HTTP/JSON

The OTLP receiver can receive trace export calls via HTTP/JSON in addition to
gRPC. The HTTP/JSON address is the same as gRPC as the protocol is recognized
and processed accordingly. Note the serialization format needs to be [protobuf JSON](https://developers.google.com/protocol-buffers/docs/proto3#json).

The HTTP/JSON configuration also provides `traces_url_path`, `metrics_url_path`, and `logs_url_path`
configuration to allow the URL paths that signal data needs to be sent to be modified per signal type.  These default to
`/v1/traces`, `/v1/metrics`, and `/v1/logs` respectively.

To write traces with HTTP/JSON, `POST` to `[address]/[traces_url_path]` for traces,
to `[address]/[metrics_url_path]` for metrics, to `[address]/[logs_url_path]` for logs.
The default port is `4318`.  When using the `otlphttpexporter` peer to communicate with this component,
use the `traces_endpoint`,  `metrics_endpoint`, and `logs_endpoint` settings in the `otlphttpexporter` to set the
proper URL to match the address and URL signal path on the `otlpreceiver`.

### CORS (Cross-origin resource sharing)

The HTTP/JSON endpoint can also optionally configure [CORS][cors] under `cors:`.
Specify what origins (or wildcard patterns) to allow requests from as
`allowed_origins`. To allow additional request headers outside of the [default
safelist][cors-headers], set `allowed_headers`. Browsers can be instructed to
[cache][cors-max-age] responses to preflight requests by setting `max_age`.

[cors]: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
[cors-headers]: https://developer.mozilla.org/en-US/docs/Glossary/CORS-safelisted_request_header
[cors-max-age]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Max-Age

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: "localhost:4318"
        cors:
          allowed_origins:
            - http://test.com
            # Origins can have wildcards with *, use * by itself to match any origin.
            - https://*.example.com
          allowed_headers:
            - Example-Header
          max_age: 7200
```

[beta]: https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[core]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol
[stable]: https://github.com/open-telemetry/opentelemetry-collector#stable
