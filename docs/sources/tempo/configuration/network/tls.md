---
title: Configure TLS communication
menuTitle: Configure TLS
description: Configure TLS for Tempo server endpoints, gRPC clients, trace receivers, object storage, and caches to secure communication between components.
aliases:
  - ../../configuration/tls/ # /docs/tempo/<TEMPO_VERSION>/configuration/tls/
---

# Configure TLS communication

Tempo can be configured to communicate between components using Transport Layer Security, or TLS.
TLS secures three categories of connections:

- Server and client: Communication between internal Tempo components (for example, querier to query-frontend).
- Receiver: Incoming trace data from instrumented applications or collectors to the distributor.
- Storage and cache: Connections to backend object storage (S3) and caches (Memcached, Redis).

{{< admonition type="note" >}}
The ciphers and TLS version here are for example purposes only. We aren't recommending which ciphers or TLS versions to use in production environments.
{{< /admonition >}}

## Server configuration

Every Tempo component exposes gRPC and HTTP endpoints. Use the `server` block to enable TLS on these endpoints.

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

Valid values for the `client_auth_type` are documented in the standard `crypto/tls` package under [`ClientAuthType`](https://pkg.go.dev/crypto/tls#ClientAuthType).

## Client configuration

Several Tempo components configure gRPC clients to communicate with other components. For example, the querier contacts the query-frontend to request work. If the server endpoint uses TLS, the corresponding client must also enable TLS.

Tempo uses a standard `grpc_client_config` stanza for each of these client connections.

You can optionally omit `tls_min_version`, `tls_cipher_suites`, and `tls_insecure_skip_verify`. Whether `tls_server_name` is required depends on your environment.

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

Set this configuration block at the following locations:

- `live_store_client.grpc_client_config`
- `querier.frontend_worker.grpc_client_config`
- `backend_scheduler_client.grpc_client_config`

Additionally, `memberlist` must also be configured, but the client configuration is nested directly under `memberlist` as follows. The same configuration options are available as above.

```yaml
memberlist:
    tls_enabled: true
    tls_cert_path: /tls/tls.crt
    tls_key_path: /tls/tls.key
    tls_ca_path: /tls/ca.crt
    tls_server_name: tempo.trace.svc.cluster.local
    tls_insecure_skip_verify: false
```

## Receiver TLS

Receiver TLS secures the connection between trace sources, such as instrumented applications, OpenTelemetry Collectors, or Grafana Alloy, and the Tempo distributor.
The receiver configuration uses the OpenTelemetry Collector TLS settings.
For the full set of options, refer to the upstream [OTLP receiver TLS documentation](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/otlpreceiver/config.md#configtls-tlsserversetting) and the [`configtls` package reference](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configtls).

### TLS fields

The following fields are available in the receiver `tls` block:

| Field | Required | Description |
|---|---|---|
| `cert_file` | Yes | Path to the TLS certificate file. |
| `key_file` | Yes | Path to the TLS private key file. |
| `ca_file` | No | Path to a CA certificate bundle used by a TLS **client** to verify the server's certificate. Use this field on the sending side (for example, in an Alloy exporter or Kafka client) to trust a custom or internal CA. On receivers, this field has no effect because receivers don't initiate outbound TLS connections. If omitted, the system root CA pool is used. |
| `client_ca_file` | No | Path to the CA certificate used by a TLS **server** to verify client certificates. When set on a receiver, it requires connecting clients to present a valid certificate signed by this CA (mTLS). Use this field to enable mTLS on receivers. |
| `min_version` | No | Minimum TLS version to accept, for example `"1.2"` or `"1.3"`. |

### Configure server-only TLS

For server-only TLS, the distributor presents its certificate and clients verify it.
Only `cert_file` and `key_file` are required:

```yaml
distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          tls:
            cert_file: /tls/tls.crt
            key_file: /tls/tls.key
            min_version: "1.2"
```

### Configure mutual TLS (mTLS)

For mutual TLS, both the server and client present certificates.
Add `client_ca_file` so the distributor requires and verifies client certificates:

```yaml
distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          tls:
            cert_file: /tls/tls.crt
            key_file: /tls/tls.key
            client_ca_file: /tls/ca.crt
            min_version: "1.2"
```

### Supported receiver TLS paths

You can set a `tls` block on the following receiver configurations:

- `distributor.receivers.otlp.protocols.grpc.tls`
- `distributor.receivers.otlp.protocols.http.tls`
- `distributor.receivers.kafka.tls` (client TLS for connecting to Kafka brokers)
- `distributor.receivers.zipkin.tls`
- `distributor.receivers.jaeger.protocols.grpc.tls`
- `distributor.receivers.jaeger.protocols.thrift_http.tls`

{{< admonition type="note" >}}
The Kafka receiver TLS block configures **client** TLS for the connection to Kafka brokers, unlike the other receiver TLS blocks which configure **server** TLS for incoming connections.
{{< /admonition >}}

### Configure the sending side

When you enable TLS on the Tempo receiver, you must also configure TLS on the client that sends traces.
The following example shows a matching Grafana Alloy configuration for an OTLP gRPC exporter with server-only TLS:

```alloy
otelcol.exporter.otlp "tempo" {
  client {
    endpoint = "tempo.trace.svc.cluster.local:4317"
    tls {
      ca_file = "/tls/ca.crt"
    }
  }
}
```

For mTLS, include the client certificate and key:

```alloy
otelcol.exporter.otlp "tempo" {
  client {
    endpoint = "tempo.trace.svc.cluster.local:4317"
    tls {
      ca_file   = "/tls/ca.crt"
      cert_file = "/tls/tls.crt"
      key_file  = "/tls/tls.key"
    }
  }
}
```

## Storage and cache TLS

### S3 and S3-compatible storage

If you use a self-managed S3-compatible backend (for example, MinIO) with a custom CA or client certificates, configure TLS on the S3 storage backend.
The TLS fields are set inline under `storage.trace.s3`:

```yaml
storage:
  trace:
    backend: s3
    s3:
      bucket: tempo-traces
      endpoint: minio.example.com:9000
      tls_ca_path: /tls/ca.crt
      tls_cert_path: /tls/tls.crt
      tls_key_path: /tls/tls.key
      tls_server_name: minio.example.com
      tls_insecure_skip_verify: false
      tls_min_version: VersionTLS12
```

GCS and Azure Blob Storage rely on their respective SDK defaults for TLS and don't expose separate TLS fields in the Tempo configuration.

### Redis cache

If you use Redis as a cache backend, enable TLS with the `tls_enabled` field.
Redis TLS has limited configuration. It supports `tls_enabled` and `tls_insecure_skip_verify` but doesn't expose CA or client certificate path fields:

```yaml
cache:
  caches:
    - redis:
        endpoint: redis.example.com:6380
        tls_enabled: true
        tls_insecure_skip_verify: false
      roles:
        - parquet-footer
        - bloom
        - frontend-search
```

## Gateway and ingress TLS

Tempo doesn't include a built-in gateway component.
If you end TLS at an ingress controller, load balancer, or reverse proxy in front of Tempo, configure TLS on that component rather than in the Tempo configuration.
When TLS is ended at the ingress, Tempo receivers don't need TLS configured for internal connections.

## Configure TLS with Helm

To configure TLS with the Helm chart, you must have a TLS key-pair and CA certificate stored in a Kubernetes secret.
The following example mounts a secret called `tempo-distributed-tls` into the pods at `/tls` and modifies the configuration of Tempo to use the files.
In this example, the Tempo components share a single TLS certificate.
The `tls_server_name` configuration must match the certificate.

```yaml
distributor:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
blockBuilder:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
liveStore:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
backendScheduler:
  extraVolumeMounts:
    - mountPath: /tls
      name: tempo-distributed-tls
  extraVolumes:
    - name: tempo-distributed-tls
      secret:
        secretName: tempo-distributed-tls
backendWorker:
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
    live_store_client:
      grpc_client_config:
        tls_ca_path: /tls/ca.crt
        tls_cert_path: /tls/tls.crt
        tls_enabled: true
        tls_key_path: /tls/tls.key
        tls_server_name: tempo-distributed.trace.svc.cluster.local
    backend_scheduler_client:
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
A relabel configuration like the following does this configuration for you dynamically.

```json
{
  source_labels: ['__meta_kubernetes_pod_annotation_prometheus_io_scheme'],
  action: 'replace',
  target_label: '__scheme__',
  regex: '(https?)',
  replacement: '$1',
},
```
