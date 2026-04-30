---
title: Configure IPv6
description: Learn how to configure IPv6 for Tempo.
menuTitle: Configure IPv6
aliases:
  - ../../configuration/ipv6/ # /docs/tempo/<TEMPO_VERSION>/configuration/ipv6/
---

# Configure IPv6

Tempo can be configured to communicate between the components using Internet Protocol Version 6, or IPv6.

{{< admonition type="note" >}}
The underlying infrastructure must support this address family. This configuration may be used in a single-stack IPv6 environment, or in a dual-stack environment where both IPv6 and IPv4 are present. In a dual-stack scenario, only one address family may be configured at a time, and all components must be configured for that address family.
{{< /admonition >}}

## Protocol configuration

This sample listen configuration will allow the gRPC and HTTP servers to listen on IPv6, and configure the various memberlist components to enable IPv6.

```yaml
memberlist:
  bind_addr:
    - '::'
  bind_port: 7946

# Required only when using the global ingestion rate strategy.
distributor:
  ring:
    enable_inet6: true

# Only needed if the backend-worker ring is enabled (kvstore.store is set).
backend_worker:
  ring:
    enable_inet6: true

metrics_generator:
  ring:
    enable_inet6: true

live_store:
  ring:
    enable_inet6: true

server:
  grpc_listen_address: '::0'
  grpc_listen_port: 9095
  http_listen_address: '::0'
  http_listen_port: 3200
```

{{< admonition type="note" >}}
The `live_store.partition_ring` doesn't expose `enable_inet6`. It inherits its address family from the top-level `memberlist` configuration.
{{< /admonition >}}

## Kubernetes service configuration

Each service fronting the workloads needs to be configured with `spec.ipFamilies` and `spec.ipFamilyPolicy` set. Refer to this `backend-worker` example:

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    name: backend-worker
  name: backend-worker
  namespace: tracing
spec:
  clusterIP: fccb::31a7
  clusterIPs:
    - fccb::31a7
  internalTrafficPolicy: Cluster
  ipFamilies:
    - IPv6
  ipFamilyPolicy: SingleStack
  ports:
    - name: backend-worker-http-metrics
      port: 3200
      protocol: TCP
      targetPort: 3200
  selector:
    app: backend-worker
    name: backend-worker
  sessionAffinity: None
  type: ClusterIP
```

In addition to the per-component services, Tempo components discover each other through a separate headless `gossip-ring` Service that exposes the memberlist port. Configure it as IPv6 so that components can join the cluster over IPv6.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: gossip-ring
  namespace: tracing
spec:
  clusterIP: None
  ipFamilies:
    - IPv6
  ipFamilyPolicy: SingleStack
  ports:
    - name: gossip-ring
      port: 7946
      protocol: TCP
      targetPort: 7946
  selector:
    tempo-gossip-member: "true"
```

Each Tempo component then joins `memberlist` by referencing the headless Service's DNS name in its `memberlist.join_members` configuration:

```yaml
memberlist:
  join_members:
    - dns+gossip-ring.tracing.svc.cluster.local.:7946
```

## Verify the listeners

The Tempo container image is distroless and doesn't include a shell or networking utilities. You can't `exec` into a Tempo Pod directly to inspect listeners.

To inspect listening sockets, attach an [ephemeral debug container](https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/#ephemeral-container) that shares the target Pod's network namespace, using a debug image that includes the `ss` utility:

```sh
kubectl debug -n tracing -it <pod-name> \
  --image=<your-debug-image> \
  --target=backend-worker \
  -- ss -ltn
```

You should see IPv6 wildcard listeners on the memberlist, gRPC, and HTTP ports:

```text
State   Recv-Q   Send-Q     Local Address:Port     Peer Address:Port
LISTEN  0        4096                   *:7946                *:*
LISTEN  0        4096                   *:9095                *:*
LISTEN  0        4096                   *:3200                *:*
```

The `*` in `Local Address` indicates the process is listening on all addresses for the configured family.
