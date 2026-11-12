# Encoding extensions

The encoding extensions can be used by compatible receivers or exporters to encode or decode data into/from a specific
format. This is useful when the data is being sent to/from a system that expects a specific format and doesn't support
the OpenTelemetry protocol.

_🚧 Under active development 🚧_

## Component Milestones

To help track what work needs to be done with this component, these are the currently active goals being
worked towards.

### Development

- Add encoding extensions support additionally to the existing ways of configuring encodings (where applicable)
  to the following components:
    - `file receiver`
    - `file exporter`
    - `kafka receiver`
    - `kafka exporter`
    - `kinesis exporter`
    - `pulsar receiver`
    - `pulsar exporter`
- Add encoding extensions for open source formats, ie: `otlp`, `zipkin`, `jaeger`
- Deprecate the previously available ways of configuring encodings (where applicable).
- Remove the previously available ways of configuring encodings in favour of using the encoding extension.

## Example configuration

```yaml
extensions:
  zipkin_encoding:
    protocol: zipkin_proto
    version: v2

receivers:
  kafka:
    traces:
      encoding: zipkin_encoding
    # ... other configuration values
```
