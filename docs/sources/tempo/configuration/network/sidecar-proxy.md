---
title: Run Tempo distributed with sidecar proxies
menuTitle: Configure sidecar proxies
description: Configure Tempo distributed to run with sidecar proxies
aliases:
  - ../../configuration/sidecar-proxy/ # /docs/tempo/<TEMPO_VERSION>/configuration/sidecar-proxy/
---

# Run Tempo distributed with sidecar proxies

You can route inter-pod gRPC traffic run through a sidecar proxy to meet requirements such as custom security, routing, or logging.
Common examples include Envoy, Nginx, Traefik, or service meshes like Istio and Linkerd.

## How Tempo pods communicate

Tempo pods communicate using gRPC.

The different components like distributors and ingesters find each other by a shared ring with the list of pods, their roles, and their addresses.
Pods advertise their address and listening port into the ring when they start, and deregister themselves when they exit.

The overall network looks like this:

![Tempo distributed network overview](/static/img/docs/tempo/sidecar-proxy/tempo-network-sidecar-proxy-simple.svg)

The low-level ring data for ingesters can be viewed by browsing to the `/ingester/ring` URL on a distributor. It looks like this:

![Ring status with default port](/static/img/docs/tempo/sidecar-proxy/screenshot-tempo-sidecar.png)

By default, gRPC traffic uses port 9095, but this can be changed by customizing the `grpc_listen_port` for each pod that needs it.

```yaml
server:
  grpc_listen_port: 12345
```

The ring contents reflect the new port:

![Ring status with updated ports](/static/img/docs/tempo/sidecar-proxy/screenshot-tempo-sidecar-proxies.png)

## Run Tempo with proxies

Some installations require that the inter-pod gRPC traffic runs through a sidecar proxy.
Running Tempo with proxies requires setting two ports for each pod: one for the Tempo process and one for the sidecar.
Additionally, the ring contents must reflect the proxy's port so that traffic from other pods goes through the proxy.

The overall network looks like this:

![Tempo distributed network overview](/static/img/docs/tempo/sidecar-proxy/tempo-network-sidecar-proxy-complex.svg)

This cannot be accomplished by setting the same `grpc_listen_port` as in the previous example. Instead, we need the ingester to _listen_ on port A but _advertise_ itself on port B. This is done by customizing the ingester's lifecycler port:

```yaml
ingester:
   lifecycler:
       port: 12345
```

Now, the ingester is listening for regular traffic on port 9095, but the distributor will route traffic to it on port 12345.

## Metrics-generator proxy

You can customize the lifecyler port in the metrics-generator. To set an instance port for the metrics-generator, use this configuration:

```yaml
metrics_generator:
  ring:
    instance_port: 12345
```

Replace `12345` with the correct port number.
