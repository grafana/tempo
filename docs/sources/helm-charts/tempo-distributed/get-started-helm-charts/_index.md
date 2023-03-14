---
description: Learn how to get started with Grafana Tempo using the Helm chart.
menuTitle: Get started
title: Get started with Grafana Tempo using the Helm chart
weight: 20
keywords:
  - Helm chart
  - Kubernetes
  - Grafana Tempo
---

# Get started with Grafana Tempo using the Helm chart

The Grafana Tempo Helm chart allows you to configure, install, and upgrade Grafana Tempo within a Kubernetes cluster. Using this procedure, you will:

- Create a custom namespace within your Kubernetes cluster
- Install Helm and the grafana helm-charts repository
- Configure Google Cloud Platform storage for traces
- Install Tempo using Helm

To learn more about Helm, read the [Helm documentation](https://helm.sh/).

If you are using Helm to install Grafana Enterprise Traces, then you will also need to ensure that you:

- have created an additional storage bucket for the admin resources
- Disable the gateway
- Enable the enterpriseGateway
- Obtain a GET license and configure the values

## Before you begin

These instructions are common across any flavor of Kubernetes. They also assume that you know how to install, configure, and operate a Kubernetes cluster.
It also assumes that you have an understanding of what the `kubectl` command does.

> **CAUTION**: This procedure is primarily aimed at local or development setups.

### Hardware requirements

- A single Kubernetes node with a minimum of 4 cores and 16GiB RAM

### Software requirements

- Kubernetes 1.20 or later (see [Kubernetes installation documentation](https://kubernetes.io/docs/setup/))
- The `kubectl` command for your version of Kubernetes
- Helm 3 or later (see [Helm installation documentation](https://helm.sh/docs/intro/install/))

Verify that you have:

- Access to the Kubernetes cluster
- Persistent storage is enabled in the Kubernetes cluster, which has a default storage class setup. You can [change the default StorageClass](https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/).
- Access to a local storage option (like MinIO) or a storage bucket like Amazon S3, Azure Blob Storage, or Google Cloud Platform (for example, [Google Cloud Storage instructions](https://cloud.google.com/storage/docs/creating-buckets))
- DNS service works in the Kubernetes cluster
- Optional: An ingress controller is set up in the Kubernetes cluster, for example [ingress-nginx](https://kubernetes.github.io/ingress-nginx/)

> **NOTE**: If you want to access Tempo from outside of the Kubernetes cluster, you may need an ingress. Ingress-related procedures are marked as optional.

<!-- This section should be verified before being made visible. It’s from Mimir and might need to be updated for Tempo.

## Security setup

This installation will not succeed if you have enabled the [PodSecurityPolicy](*https://v1-23.docs.kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#podsecuritypolicy) admission controller or if you are enforcing the Restricted policy with [Pod Security](https://v1-24.docs.kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-admission-labels-for-namespaces) admission controller. The reason is that the installation includes a deployment of MinIO. The [minio/minio chart](https://github.com/minio/minio/tree/master/helm/minio) is not compatible with running under a Restricted policy or the PodSecurityPolicy that the mimir-distributed chart provides.

If you are using the PodSecurityPolicy admission controller, then it is not possible to deploy the mimir-distributed chart with MinIO. Refer to Run Grafana Mimir in production using the Helm chart for instructions on setting up an external object storage and disable the built-in MinIO deployment with minio.enabled: false in the Helm values file.

If you are using the Pod Security admission controller, then MinIO and the tempo-distributed chart can successfully deploy under the baseline pod security level.
-->

## Create a custom namespace and add the Helm repository

Using a custom namespace solves problems later on because you do not have to overwrite the default namespace.

1. Create a unique Kubernetes namespace, for example `tempo-test`, and switch your local context to use it:

```bash
kubectl create namespace tempo-test
kubens tempo-test
```

For more details, see the Kubernetes documentation about [Creating a new namespace](https://kubernetes.io/docs/tasks/administer-cluster/namespaces/#creating-a-new-namespace).

1. Set up a Helm repository using the following commands:

   ```bash
   helm repo add grafana https://grafana.github.io/helm-charts
   helm repo update
   ```

   > **NOTE**: The Helm chart at https://grafana.github.io/helm-charts is a publication of the source code at grafana/tempo.

## Set Helm chart values

Your Helm chart values are set in the `custom.yaml` file. The following example `custom.yaml` file sets the storage and traces options, enables the gateway, and sets the cluster to main. The `traces` configure the distributor's receiver protocols.

Next, you will:

1. Create a `custom.yaml` file
2. Set your storage values, the example above points to the MinIO instance configured by the chart
3. Set your traces values to configure the receivers on the Tempo distributor

### Tempo helm chart values

```yaml
---
storage:
  trace:
    backend: s3
    s3:
      access_key: 'grafana-tempo'
      secret_key: 'supersecret'
      bucket: 'tempo-traces'
      endpoint: 'tempo-minio:9000'
      insecure: true
minio:
  enabled: true
  mode: standalone
  rootUser: grafana-tempo
  rootPassword: supersecret
  buckets:
    # Default Tempo storage bucket.
    - name: tempo-traces
      policy: none
      purge: false
traces:
  otlp:
    grpc:
      enabled: true
    http:
      enabled: true
  zipkin:
    enabled: false
  jaeger:
    thriftHttp:
      enabled: false
  opencensus:
    enabled: false
```

### Grafana Enterprise Traces helm chart values

The values in the example below provide configuration values for GET.
These values include an additional `admin` bucket, the `gateway` has been disabled, the `enterpriseGateway` has been enabled, and a license has been specified.

```yaml
---
global:
  clusterDomain: 'cluster.local'

multitenancyEnabled: true
enterprise:
  enabled: true
  image:
    tag: v2.0.1
enterpriseGateway:
  enabled: true
gateway:
  enabled: false
minio:
  enabled: true
storage:
  trace:
    backend: s3
    s3:
      access_key: 'grafana-tempo'
      secret_key: 'supersecret'
      bucket: 'enterprise-traces'
      endpoint: 'tempo-minio:9000'
      insecure: true
  admin:
    backend: s3
    s3:
      access_key_id: 'grafana-tempo'
      secret_access_key: 'supersecret'
      bucket_name: 'enterprise-traces-admin'
      endpoint: 'tempo-minio:9000'
      insecure: true
traces:
  otlp:
    http:
      enabled: true
    grpc:
      enabled: true
distributor:
  config:
    log_received_spans:
      enabled: true

license:
  contents: |
    LICENSEGOESHERE
```

#### Enterprise license configuration

If you are using GET, you need to configure a license, by adding the license to the `custom.yaml` file or by using a secret that contains the license.
Only one of these options should be used. 

> **NOTE**: The [Set up GET instructions](https://grafana.com/docs/enterprise-traces/latest/setup/#obtain-a-get-license) explain how to obtain a license.

Using the first option, you can specify the license text in the `custom.yaml` values file created above, in the `license:` section.

```yaml
license:
  contents: |
    LICENSEGOESHERE
```

If you do not with to specific the license in the `custom.yaml` file, you can use a secret that contains the license content that is referenced.

1. Create the secret.

   ```bash
   kubectl create secret generic tempo-license --from-file=license.jwt
   ```

1. Configure the `custom.yaml` that you created above to reference the secret.

   ```yaml
   license:
     external: true
   ```

### Set your storage option

Before you run the Helm chart, you need to configure where trace data will be stored.

The `storage` block defined in the `values.yaml` file is used to configure the storage that Tempo uses for trace storage.

The procedure below configures MinIO as the local storage option managed by the Helm chart.
However, you can use a other storage provides. Refer to the Optional storage section below.

1. Update the configuration options in `custom.yaml` for your configuration.

   ```yaml
   ---
   storage:
     trace:
       backend: s3
       s3:
         access_key: 'grafana-tempo'
         secret_key: 'supersecret'
         bucket: 'tempo-traces'
         endpoint: 'tempo-minio:9000'
         insecure: true
   ```

   Enterprise users will also need to specify an additional bucket for `admin` resources.

    ```yaml
    storage:
      admin:
        backend: s3
        s3:
          access_key_id: 'grafana-tempo'
          secret_access_key: 'supersecret'
          bucket_name: 'enterprise-traces-admin'
          endpoint: 'tempo-minio:9000'
          insecure: true
    ```

1. Optional: Locate the MinIO section and change the the username and password to something you wish to use.

   ```yaml
   minio:
     enabled: true
     mode: standalone
     rootUser: minio
     rootPassword: minio123
   ```

### Optional: Other storage options

Each storage provider has a different configuration stanza, which are detailed in Tempo's documentation. You will need to update your configuration based upon you storage provider.
Refer to the [`storage` configuration block](https://grafana.com/docs/tempo/latest/configuration/#storage) for information on storage options.

To use other storage options, set `minio.enabled: false` in the `values.yaml` file:

```yaml
---
minio:
  enabled: false # Disables the MinIO chart
```

Update the `storage` configuration options based upon your requirements:

- [Amazon S3 configuration documentation](https://grafana.com/docs/tempo/latest/configuration/s3/). The Amazon S3 example is identical to the MinIO configuration. The two last options, `endpoint` and `insecure`, are dropped.

- [Azure Blob Storage configuration documentation](https://grafana.com/docs/tempo/latest/configuration/azure/)

- [Google Cloud Storage configuration documentation](https://grafana.com/docs/tempo/latest/configuration/gcs/)

### Set traces receivers

Tempo can be configured to receive data from OTLP, Jaegar, Zipkin, Kafka, and OpenCensus.
The following example enables OTLP on the distributor. For other options, refer to the [distributor documentation](https://grafana.com/docs/tempo/latest/configuration/#distributor)

The example used in this procedure has OTLP enabled.

Enable any other protocols based on your requirements.

```yaml
traces:
  otlp:
    grpc:
      enabled: true
    http:
      enabled: true
  zipkin:
    enabled: false
  jaeger:
    thriftHttp:
      enabled: false
  opencensus:
    enabled: false
```

### Optional: Add custom configurations

You can use a YAML file, like `custom.yaml`, to store custom configuration options that override the defaults present in the Helm chart.
The [tempo-distributed Helm chart's README](https://github.com/grafana/helm-charts/blob/main/charts/tempo-distributed/README.md) contains a list of available options.
The `values.yaml` files provides the defaults for the helm chart. 

To see all of the configurable parameters for the `tempo-distributed` Helm chart, use the following command:

```bash
helm show values grafana/tempo-distributed
```

The configuration sections are added to the `custom.yaml` file. This file is included when you install or upgrade the Helm chart.

#### Optional: Configure an ingress

An ingress lets you externally access a Kubernetes cluster.
Replace `<ingress-host>` with a suitable hostname that DNS can resolve to the external IP address of the Kubernetes cluster.
For more information, see [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

> **NOTE**: On Linux systems, and if it is not possible for you set up local DNS resolution, you can use the `--add-host=<ingress-host>:<kubernetes-cluster-external-address>` command-line flag to define the `<ingress-host>` local address for the docker commands in the examples that follow.

1. Open your `custom.yaml` or create a YAML file of Helm values called `custom.yaml`.
1. Add the following configuration to the file:
   ```yaml
   nginx:
     ingress:
       enabled: true
       ingressClassName: nginx
       hosts:
         - host: <ingress-host>
           paths:
             - path: /
               pathType: Prefix
       tls: {} # empty, disabled.
   ```
1. Save the changes.

## Install Grafana Tempo using the Helm chart

Use the following command to install Tempo using the configuration options you’ve specified in the `custom.yaml` file:

```bash
helm -n tempo-test install tempo grafana/tempo-distributed -f custom.yaml
```

> **NOTE**: The output of the command contains the write and read URLs necessary for the following steps.

If the installation is successful, the output will be similar to this:

```yaml
>  helm -n tempo-test install tempo grafana/tempo-distributed -f custom.yaml

W0210 15:02:09.901064    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.904082    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.906932    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.929946    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.930379    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
NAME: tempo
LAST DEPLOYED: Fri Feb 10 15:02:08 2023
NAMESPACE: tempo-test
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
***********************************************************************
 Welcome to Grafana Tempo
 Chart version: 1.0.1
 Tempo version: 2.0.0
***********************************************************************

Installed components:
* ingester
* distributor
* querier
* query-frontend
* compactor
* memcached
```

> **NOTE**: If you update your `values.yaml` or `custom.yaml`, run the same helm install command and replace `install` with `upgrade`.

Check the statuses of the Tempo pods:

```bash
kubectl -n tempo-test get pods
```

The results look similar to this:

```
NAME                                    READY   STATUS    RESTARTS   AGE
tempo-compactor-86cd974cf-8qrk2         1/1     Running   0          22h
tempo-distributor-bbf4889db-v8l8r       1/1     Running   0          22h
tempo-ingester-0                        1/1     Running   0          22h
tempo-ingester-1                        1/1     Running   0          22h
tempo-ingester-2                        1/1     Running   0          22h
tempo-memcached-0                       1/1     Running   0          8d
tempo-minio-6c4b66cb77-sgm8z            1/1     Running   0          26h
tempo-querier-777c8dcf54-fqz45          1/1     Running   0          22h
tempo-query-frontend-7f7f686d55-xsnq5   1/1     Running   0          22h
```

Wait until all of the pods have a status of Running or Completed, which might take a few minutes.

For Enterprise users, the output results look similar to this:

```bash
❯ k get pods
NAME                                        READY   STATUS      RESTARTS      AGE
tempo-admin-api-7c59c75f6c-wvj75            1/1     Running     0             86m
tempo-compactor-75777b5d8c-5f44z            1/1     Running     0             86m
tempo-distributor-94fd965f4-prkz6           1/1     Running     0             86m
tempo-enterprise-gateway-6d7f78cf97-dhz9b   1/1     Running     0             86m
tempo-ingester-0                            1/1     Running     0             86m
tempo-ingester-1                            1/1     Running     1 (86m ago)   86m
tempo-ingester-2                            1/1     Running     1 (86m ago)   86m
tempo-memcached-0                           1/1     Running     0             86m
tempo-minio-6c4b66cb77-wjfpf                1/1     Running     0             86m
tempo-querier-6cb474546-cwlkz               1/1     Running     0             86m
tempo-query-frontend-6d6566cbf7-pcwg6       1/1     Running     0             86m
tempo-tokengen-job-58jhs                    0/1     Completed   0             86m
```

Note that the `tempo-tokengen-job` has emitted a log message containing the initial admin token.

Retrieve the token with this command:

```
kubectl get pods | awk '/.*-tokengen-job-.*/ {print $1}' | xargs -I {} kubectl logs {} | awk '/Token:\s+/ {print $2}'
```

To get the logs for the `tokengen` pod, you can use:

```
kubectl logs tempo-tokengen-job-58jhs
```

## Next step

The next step is to test your Tempo installation by sending trace data to Grafana. You can use the [Set up a test application for a Tempo cluster](https://grafana.com/docs/tempo/latest/setup/set-up-test-app/) document for step-by-step instructions.

If you already have Grafana available, you can add a Tempo data source using the URL fitting to your environment. For example:
`http://tempo-query-frontend.trace-test.svc.cluster.local:3100`

Enterprise users may wish to [install the Enterprise Traces plugin](https://grafana.com/docs/enterprise-traces/latest/setup/setup-get-plugin-grafana/) in their Grafana Enterprise instance to allow configuration of tenants, tokens, and access policies. Once a user, and access policy have been created using the plugin, a datasource can be configured to point at `http://tempo-enterprise-gateway.tempo-test.svc.cluster.local:3100`.
