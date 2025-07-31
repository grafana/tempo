---
title: 'Quickstart'
description: Quickstart to deploy Tempo with the Tempo Operator
menuTitle: Quickstart
weight: 100
aliases:
  - ../../../../../operator/quickstart/ # /docs/tempo/next/operator/quickstart/
  - ../../../../../setup/operator/quickstart/ # /docs/tempo/next/setup/operator/quickstart/
---

# Quickstart

One page summary on how to start with Tempo Operator and `TempoStack`.

## Requirements

The easiest way to start with the Tempo Operator is to use Kubernetes [kind](https://kind.sigs.k8s.io/).

## Deploy

To install the operator in an existing cluster, make sure you have [cert-manager](https://cert-manager.io/docs/installation/) installed and run:

```shell
kubectl apply -f https://github.com/grafana/tempo-operator/releases/latest/download/tempo-operator.yaml
```

Once you have the operator deployed you need to install a storage backend. For this quick start guide, we will install [`MinIO`](https://min.io/) as follows:

```shell
kubectl apply -f https://raw.githubusercontent.com/grafana/tempo-operator/main/minio.yaml
```

After minio was deployed, create a secret for MinIO in the namespace you are using:

```yaml
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: minio-test
stringData:
  endpoint: http://minio.minio.svc:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF
```

Then create Tempo CR:

```yaml
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: minio-test
      type: s3
  storageSize: 1Gi
  resources:
    total:
      limits:
        memory: 2Gi
        cpu: 2000m
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
EOF
```

After create the `TempoStack` CR, you should see a some pods on the namespace. Wait for the stack to stabilize.

The stack deployed above is configured to receive Jaeger, Zipkin, and OpenTelemetry (OTLP) protocols.
Because the Jaeger Query is enabled, you can also use the Jaeger UI to inspect the data.

To do a quick test, deploy a Job that generates some traces.

```yaml
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: tracegen
spec:
  template:
    spec:
      containers:
        - name: tracegen
          image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/tracegen:latest
          command:
            - "./tracegen"
          args:
            - -otlp-endpoint=tempo-simplest-distributor:4317
            - -otlp-insecure
            - -duration=30s
            - -workers=1
      restartPolicy: Never
  backoffLimit: 4
EOF
```

Forward the Jaeger Query port to see the traces:

```
kubectl port-forward svc/tempo-simplest-query-frontend 16686:16686
```

Visit http://localhost:16686 to view the results.
