---
title: Run Tempo distributed with sidecar proxies
menuTitle: Configure sidecar proxies
description: Configure Tempo distributed to run with sidecar proxies
aliases:
  - ../../configuration/sidecar-proxy/ # /docs/tempo/<TEMPO_VERSION>/configuration/sidecar-proxy/
---

# Run Tempo distributed with sidecar proxies

You can route inter-pod gRPC traffic through a sidecar proxy to meet requirements such as custom security, routing, or logging.
Common examples include Envoy, Nginx, Traefik, or service meshes like Istio and Linkerd.

This page applies only to the [microservices deployment mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/).
In monolithic (single-binary) mode, components communicate in-process, so there's no inter-pod gRPC traffic for a sidecar to proxy.

## How Tempo pods communicate

In microservices mode, Tempo's write path runs through Kafka and the read path uses gRPC.
Distributors write to Kafka, and live-stores, block-builders, and metrics-generators each consume from Kafka independently.
Queriers contact live-stores over gRPC for recent data.

For background on how each component communicates, refer to:

- [Deployment modes](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/) for the overall network model.
- [Live-store](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/live-store/) and [Querier](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/querier/) for the gRPC read path that this page covers.

This page focuses on the querier-to-live-store gRPC traffic, which a sidecar proxy can wrap.
Queriers discover live-stores through a shared hash ring: live-stores register their address and port in the ring at startup and deregister on exit.

The overall network looks like this:

![Tempo distributed network overview](/static/img/docs/tempo/sidecar-proxy/tempo-network-sidecar-proxy-simple.svg)

You can view the low-level ring data by browsing to the `/live-store/ring` URL on any component that imports the live-store ring, for example a distributor or querier. It looks like this:

![Ring status with default port](/static/img/docs/tempo/sidecar-proxy/screenshot-tempo-sidecar.png)

By default, gRPC traffic uses port 9095, but you can change it by customizing `server.grpc_listen_port` on each pod that needs it.
This setting changes the gRPC port for the entire process, so make sure any other components that connect to the pod, for example queriers connecting to a query-frontend, use the new port too.

```yaml
server:
  grpc_listen_port: 12345
```

The ring contents reflect the new port:

![Ring status with updated ports](/static/img/docs/tempo/sidecar-proxy/screenshot-tempo-sidecar-proxies.png)

## Run Tempo with proxies

Some installations require that the inter-pod gRPC traffic runs through a sidecar proxy.
Running Tempo with proxies requires setting two ports for the live-store: one for the live-store process and one for the sidecar.
Additionally, the live-store ring contents must reflect the proxy's port so that traffic from queriers goes through the proxy.

The overall network looks like this:

![Tempo distributed network with sidecar proxies](/static/img/docs/tempo/sidecar-proxy/tempo-network-sidecar-proxy-complex.svg)

You can't accomplish this by setting the same `grpc_listen_port` as in the previous example. Instead, the live-store needs to _listen_ on port A but _advertise_ itself on port B. To do this, customize the live-store's ring instance port:

```yaml
live_store:
  ring:
    instance_port: 12345
```

The live-store now listens for regular traffic on port 9095, but queriers route traffic to it on port 12345.

## Other components

Only the live-store ring's `instance_port` is relevant for this sidecar proxy pattern.
Other components have their own rings, but those rings aren't used by queriers for gRPC routing:

- The metrics-generator consumes trace data from Kafka in microservices mode, so distributors don't reach it through a hash ring. Refer to the [metrics-generator architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/metrics-generator/).
- Distributors and block-builders also use Kafka for the write path. Refer to the [distributor](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/distributor/) and [partition ring](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/partition-ring/) documentation.
- For other gRPC connections, such as querier-to-query-frontend, set `server.grpc_listen_port` on the destination pod and update the matching client address on the calling component.
