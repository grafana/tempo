---
title: Configure IPv6
description: Learn how to configure IPv6 for Tempo.
weight: 37
---

# Configure IPv6

Tempo can be configured to communicate between the components using Internet Protocol Version 6, or IPv6.

> Note: In order to support this support this configuration, the underlying infrastructure must support this address family. This configuration may be used in a single-stack IPv6 environment, or in a dual-stack environment where both IPv6 and IPv4 are present. In a dual-stack scenario, only one address family may be configured at a time, and all components must be configured for that address family.

## Protocol configuration

This sample listen configuration will allow the gRPC and HTTP servers to listen on IPv6, and configure the various memberlist components to enable IPv6.

```yaml
memberlist:
  bind_addr:
    - '::'
  bind_port: 7946

compactor:
  ring:
    kvstore:
      store: memberlist
    enable_inet6: true

metrics_generator:
  ring:
    enable_inet6: true

ingester:
  lifecycler:
    address: '::'
    enable_inet6: true

server:
  grpc_listen_address: '::0'
  grpc_listen_port: 9095
  http_listen_address: '::0'
  http_listen_port: 3200
```

## Kubernetes service configuration

Each service fronting the workloads will need to be configured with with `spec.ipFamilies` and `spec.ipFamilyPolicy` set. See this `compactor` example.

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    name: compactor
  name: compactor
  namespace: trace
spec:
  clusterIP: fccb::31a7
  clusterIPs:
    - fccb::31a7
  internalTrafficPolicy: Cluster
  ipFamilies:
    - IPv6
  ipFamilyPolicy: SingleStack
  ports:
    - name: compactor-http-metrics
      port: 3200
      protocol: TCP
      targetPort: 3200
  selector:
    app: compactor
    name: compactor
  sessionAffinity: None
  type: ClusterIP
```

You can check the listening service from within a pod.

```sh
❯ k exec -it compactor-55c778b8d9-2kch2 -- sh
/ # apk add iproute2
OK: 12 MiB in 27 packages
/ # ss -ltn -f inet
State   Recv-Q   Send-Q     Local Address:Port     Peer Address:Port  Process
/ # ss -ltn -f inet6
State    Recv-Q   Send-Q     Local Address:Port     Peer Address:Port  Process
LISTEN   0        4096                   *:7946                *:*
LISTEN   0        4096                   *:9095                *:*
LISTEN   0        4096                   *:3200                *:*
```
