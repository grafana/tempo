---
title: Configure TLS communication
menuTitle: Configure TLS
description: Configure Tempo components to communicate using TLS.
aliases:
  - ../../configuration/tls/ # /docs/tempo/<TEMPO_VERSION>/configuration/tls/
---

# Configure TLS communication

Tempo can be configured to communicate between the components using Transport Layer Security, or TLS.

{{< admonition type="note" >}}
The ciphers and TLS version here are for example purposes only. We are not recommending which ciphers or TLS versions for use in production environments.
{{< /admonition >}}

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
Addition TLS configuration of OTEL components can be found [here](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configtls).

An example `tls` block might look like the following:

```yaml
tls:
  ca_file: /tls/ca.crt
  cert_file: /tls/tls.crt
  key_file: /tls/tls.key
  min_version: "1.2"
```

The above structure can be set on the following receiver configurations:

- `distributor.receivers.otlp.protocols.grpc.tls`
- `distributor.receivers.otlp.protocols.http.tls`
- `distributor.receivers.zipkin.tls`
- `distributor.receivers.jaeger.protocols.grpc.tls`
- `distributor.receivers.jaeger.protocols.thrift_http.tls`

### Configure TLS with Helm

To configure TLS with the Helm chart, you must have a TLS key-pair and CA certificate stored in a Kubernetes secret.
The following example mounts a secret called `tempo-distributed-tls` into the pods at `/tls` and modifies the configuration of Tempo to make use of the files.
In this example, the Tempo components share a single TLS certificate.
Note that the `tls_server_name` configuration must match the certificate.

```yaml
compactor:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
distributor:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
ingester:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
memcached:
  extraArgs:
    - -Z
    - -o
    - ssl_chain_cert=/tls/tls.crt,ssl_key=/tls/tls.key
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
metricsGenerator:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
querier:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
queryFrontend:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
tempo:
  readinessProbe:
    httpGet:
      scheme: HTTPS
  structuredConfig:
    memberlist:
      tls_ca_path: /tls/ca.crt
      tls_cert_path: /tls/tls.crt
      tls_enabled: true
      tls_key_path: /tls/tls.key
      tls_server_name: tempo-distributed.trace.svc.cluster.local
    distributor:
      receivers:
        otlp:
          protocols:
            grpc:
              tls:
                ca_file: /tls/ca.crt
                cert_file: /tls/tls.crt
                key_file: /tls/tls.key
    ingester_client:
      grpc_client_config:
        tls_ca_path: /tls/ca.crt
        tls_cert_path: /tls/tls.crt
        tls_enabled: true
        tls_key_path: /tls/tls.key
        tls_server_name: tempo-distributed.trace.svc.cluster.local
    cache:
      caches:
        - memcached:
            consistent_hash: true
            host: tempo-distributed-memcached
            service: memcached-client
            timeout: 500ms
            tls_ca_path: /tls/ca.crt
            tls_cert_path: /tls/tls.crt
            tls_enabled: true
            tls_key_path: /tls/tls.key
            tls_server_name: tempo-distributed.trace.svc.cluster.local
          roles:
            - parquet-footer
            - bloom
            - frontend-search
    metrics_generator_client:
      grpc_client_config:
        tls_ca_path: /tls/ca.crt
        tls_cert_path: /tls/tls.crt
        tls_enabled: true
        tls_key_path: /tls/tls.key
        tls_server_name: tempo-distributed.trace.svc.cluster.local
    querier:
      frontend_worker:
        grpc_client_config:
          tls_ca_path: /tls/ca.crt
          tls_cert_path: /tls/tls.crt
          tls_enabled: true
          tls_key_path: /tls/tls.key
          tls_server_name: tempo-distributed.trace.svc.cluster.local
    server:
      grpc_tls_config:
        cert_file: /tls/tls.crt
        client_auth_type: VerifyClientCertIfGiven
        client_ca_file: /tls/ca.crt
        key_file: /tls/tls.key
      http_tls_config:
        cert_file: /tls/tls.crt
        client_auth_type: VerifyClientCertIfGiven
        client_ca_file: /tls/ca.crt
        key_file: /tls/tls.key
traces:
  otlp:
    grpc:
      enabled: true
```

Refer to the [`prometheus.scrape` docs for Alloy](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/prometheus/prometheus.scrape/) to configure TLS on the scrape.
A relabel configuration like the following will do this configuration for you dynamically.

```json
{
  source_labels: ['__meta_kubernetes_pod_annotation_prometheus_io_scheme'],
  action: 'replace',
  target_label: '__scheme__',
  regex: '(https?)',
  replacement: '$1',
},
```
