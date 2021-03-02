# TLS Configuration Settings

Crypto TLS exposes a [variety of settings](https://godoc.org/crypto/tls).
Several of these settings are available for configuration within individual
receivers or exporters.

Note that mutual TLS (mTLS) is also supported.

## TLS / mTLS Configuration

By default, TLS is enabled:

- `insecure` (default = false): whether to enable client transport security for
  the exporter's gRPC connection. See
  [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).

As a result, the following parameters are also required:

- `cert_file`: Path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
- `key_file`: Path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.

A certificate authority may also need to be defined:

- `ca_file`: Path to the CA cert. For a client this verifies the server
  certificate. For a server this verifies client certificates. If empty uses
  system root CA. Should only be used if `insecure` is set to false.

Additionally you can configure TLS to be enabled but skip verifying the server's
certificate chain. This cannot be combined with `insecure` since `insecure`
won't use TLS at all.

- `insecure_skip_verify` (default = false): whether to skip verifying the
  certificate or not.

How TLS/mTLS is configured depends on whether configuring the client or server.
See below for examples.

## Client Configuration

[Exporters](https://github.com/open-telemetry/opentelemetry-collector/blob/master/exporter/README.md)
leverage client configuration.

Note that client configuration supports TLS configuration, however
configuration parameters are not defined under `tls_settings` like server
configuration. For more information, see [configtls
README](../configtls/README.md).

Beyond TLS configuration, the following setting can optionally be configured:

- `server_name_override`: If set to a non-empty string, it will override the
  virtual host name of authority (e.g. :authority header field) in requests
  (typically used for testing).

Example:

```yaml
exporters:
  otlp:
    endpoint: myserver.local:55690
    insecure: false
    ca_file: server.crt
    cert_file: client.crt
    key_file: client.key
  otlp/insecure:
    endpoint: myserver.local:55690
    insecure: true
  otlp/secure_no_verify:
    endpoint: myserver.local:55690
    insecure: false
    insecure_skip_verify: true
```

## Server Configuration

[Receivers](https://github.com/open-telemetry/opentelemetry-collector/blob/master/receiver/README.md)
leverage server configuration.

Beyond TLS configuration, the following setting can optionally be configured
(required for mTLS):

- `client_ca_file`: Path to the TLS cert to use by the server to verify a
  client certificate. (optional) This sets the ClientCAs and ClientAuth to
  RequireAndVerifyClientCert in the TLSConfig. Please refer to
  https://godoc.org/crypto/tls#Config for more information.

Example:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: mysite.local:55690
        tls_settings:
          cert_file: server.crt
          key_file: server.key
  otlp/mtls:
    protocols:
      grpc:
        client_ca_file: client.pem
        endpoint: mysite.local:55690
        tls_settings:
          cert_file: server.crt
          key_file: server.key
  otlp/notls:
    protocols:
      grpc:
        endpoint: mysite.local:55690
```
