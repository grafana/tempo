---
title: Enable multi-tenancy
description: Enable multi-tenancy
menuTitle: Enable multi-tenancy
weight: 250
aliases:
- /docs/tempo/operator/multitenancy
---

# Enable multi-tenancy

Tempo is a multi-tenant distributed tracing backend. It supports multi-tenancy through the use of a header: `X-Scope-OrgID`.
Refer to [multi-tenancy docs]({{< relref "../../operations/multitenancy" >}}) for more details. 
This document outlines how to deploy and use multi-tenant Tempo with the Operator.

## Multi-tenancy without authentication

The following Kubernetes Custom Resource (CR) deploys a multi-tenant Tempo instance.

{{% admonition type="note" %}}
Jaeger query is not tenant aware and therefore is not supported in this configuration.
{{% /admonition %}}

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  tenants: {}
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
```

## OIDC authentication with static RBAC

On Kubernetes, a multi-tenant Tempo instance uses OIDC authentication and static RBAC authorization defined in the CR.
The instance should be accessed through service `tempo-simplest-gateway`, which handles authentication and authorization.
The service exposes Jaeger query API and OpenTelemetry gRPC (OTLP) for trace ingestion.
The Jaeger UI can be accessed at `http://<exposed gateway service>:8080/api/traces/v1/<tenant-name>/search`.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
    gateway:
      enabled: true
  storage:
    secret:
      type: s3
      name: minio-test
  storageSize: 200M
  tenants:
    mode: static
    authentication:
      - tenantName: test-oidc
        tenantId: test-oidc
        oidc:
          issuerURL: http://dex.default.svc.cluster.local:30556/dex
          redirectURL: http://tempo-simplest-gateway.default.svc.cluster.local:8080/oidc/test-oidc/callback
          usernameClaim: email
          secret:
            name: oidc-test
    authorization:
      roleBindings:
      - name: "test"
        roles:
        - read-write
        subjects:
        - kind: user
          name: "admin@example.com"
      roles:
      - name: read-write
        permissions:
        - read
        - write
        resources:
        - traces
        tenants:
        - test-oidc
```

* The secret `oidc-test` defines fields `clientID`, `clientSecret` and `issuerCAPath`.
* The RBAC gives tenant `test-oidc` read and write access for traces. 

## OpenShift

On OpenShift, the authentication and authorization does not require any third-party service dependencies.
The authentication uses OpenShift OAuth (the user is redirected to the OpenShift login page) and authorization is handled through `SubjectAccessReview` (SAR).

The instance should be accessed through service `tempo-simplest-gateway`, which handles authentication and authorization.
The service exposes Jaeger query API and OpenTelemetry gRPC (OTLP) for trace ingestion.
The Jaeger UI can be accessed at `http://<exposed gateway service>:8080/api/traces/v1/<tenant-name>/search`.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind:  TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: object-storage
      type: s3
  storageSize: 1Gi
  tenants:
    mode: openshift
    authentication:
      - tenantName: dev
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
      - tenantName: prod
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
  template:
    gateway:
      enabled: true
    queryFrontend:
      jaegerQuery:
        enabled: true
```

`ClusterRole` and `ClusterRoleBinding` objects have to be created to enable reading and writing the data.

### RBAC for reading the data

The following RBAC gives authenticated users access to read trace data for `dev` and `prod` tenants.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tempostack-traces-reader
rules:
  - apiGroups:
      - 'tempo.grafana.com'
    resources:
      - dev
      - prod
    resourceNames:
      - traces
    verbs:
      - 'get'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tempostack-traces-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tempostack-traces-reader
subjects:
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: system:authenticated
```

### RBAC for writing data

The following RBAC gives service account `otel-collector` write access for trace data for `dev` tenant.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otel-collector
  namespace: otel
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tempostack-traces-write
rules:
  - apiGroups:
      - 'tempo.grafana.com'
    resources:
      - dev
    resourceNames:
      - traces
    verbs:
      - 'create'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tempostack-traces
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tempostack-traces-write
subjects:
  - kind: ServiceAccount
    name: otel-collector
    namespace: otel
```

OpenTelemetry collector CR configuration with authentication for dev tenant.

```yaml
spec:
    serviceAccount: otel-collector
    config: |
        extensions:
          bearertokenauth:
            filename: "/var/run/secrets/kubernetes.io/serviceaccount/token"
        exporters:
          # Export the dev tenant traces to a Tempo instance
          otlp/dev:
            endpoint: tempo-simplest-gateway.tempo.svc.cluster.local:8090
            tls:
              insecure: false
              ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
            auth:
              authenticator: bearertokenauth
            headers:
              X-Scope-OrgID: "dev"
```
