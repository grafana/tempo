---
title: Configure TLS communication
menuTitle: Configure TLS
description: Configure Tempo components to communicate using TLS.
aliases:
  - ../../configuration/tls/ # /docs/tempo/<TEMPO_VERSION>/configuration/tls/
---

# Configure TLS communication

Tempo can be configured to communicate between the components using Transport Layer Security, or TLS.

{{% admonition type="note" %}}
The ciphers and TLS version here are for example purposes only. We are not recommending which ciphers or TLS versions for use in production environments.
{{% /admonition %}}

## Server configuration

This sample TLS server configuration shows supported options.

```yaml
server:
  tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  tls_min_version: VersionTLS12

  grpc_tls_config:
    cert_file: /tls/tls.crt
    key_file: /tls/tls.key
    client_auth_type: VerifyClientCertIfGiven
    client_ca_file: /tls/ca.crt
  http_tls_config:
    cert_file: /tls/tls.crt
    key_file: /tls/tls.key
    client_auth_type: VerifyClientCertIfGiven
    client_ca_file: /tls/ca.crt
```

Valid values for the `client_auth_type` are documented in the standard `crypt/tls` package under `ClientAuthType` [here](https://pkg.go.dev/crypto/tls#ClientAuthType).

## Client configuration

Several components of Tempo need to configure the gRPC clients they use to communicate with other components. For example, when the `querier` contacts the `query-frontend` to request work, the client in use must enable TLS if the server is serving a TLS endpoint.

The Tempo configuration uses a standard configuration stanza for each of these client configurations. Below is an example of the configuration.

The optional configuration elements `tls_min_version`, `tls_cipher_suites`, and `tls_insecure_skip_verify` may be omitted. The option `tls_server_name` may or may not be required, depending on the environment.

```yaml
grpc_client_config:
  tls_enabled: true
  tls_cert_path: /tls/tls.crt
  tls_key_path: /tls/tls.key
  tls_ca_path: /tls/ca.crt
  tls_server_name: tempo.trace.svc.cluster.local
  tls_insecure_skip_verify: false
  tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  tls_min_version: VersionTLS12
```

The configuration block needs to be set at the following configuration locations.

- `ingester_client.grpc_client_config`
- `metrics_generator_client.grpc_client_config`
- `querier.query-frontend.grpc_client_config`

Additionally, `memberlist` must also be configured, but the client configuration is nested directly under `memberlist` as follows. The same configuration options are available as above.

```
memberlist:
    tls_enabled: true
    tls_cert_path: /tls/tls.crt
    tls_key_path: /tls/tls.key
    tls_ca_path: /tls/ca.crt
    tls_server_name: tempo.trace.svc.cluster.local
    tls_insecure_skip_verify: false
```

### Receiver TLS

Additional receiver configuration can be added to support TLS communication for traces being sent to Tempo. The receiver configuration is pulled in from the Open Telemetry collector, and is [documented upstream here](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/otlpreceiver/config.md#configtls-tlsserversetting).

An example `tls` block might look like the following:

```yaml
tls:
  ca_file: /tls/ca.crt
  cert_file: /tls/tls.crt
  key_file: /tls/tls.key
  min_version: VersionTLS12
```

The above structure can be set on the following receiver configurations:

- `distributor.receivers.otlp.protocols.grpc.tls`
- `distributor.receivers.otlp.protocols.http.tls`
- `distributor.receivers.zipkin.tls`
- `distributor.receivers.jaeger.protocols.grpc.tls`
- `distributor.receivers.jaeger.protocols.thrift_http.tls`
