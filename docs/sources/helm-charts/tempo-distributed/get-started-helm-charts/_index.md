---
description: Learn how to get started with Grafana Tempo using the Helm chart.
menuTitle: Get started
title: Get started with Grafana Tempo using the Helm chart
weight: 20
keywords:
  - Helm chart
  - Kubernetes
  - Grafana Tempo
  - Grafana Enterprise Traces
---

# Get started with Grafana Tempo using the Helm chart

The `tempo-distributed` Helm chart allows you to configure, install, and upgrade Grafana Tempo or Grafana Enterprise Traces (GET) within a Kubernetes cluster.
Using this procedure, you need to:

- Create a custom namespace within your Kubernetes cluster
- Install Helm and the Grafana `helm-charts` repository
- Configure a storage option for traces
- Install Tempo or GET using Helm

To learn more about Helm, read the [Helm documentation](https://helm.sh/).

If you are using Helm to install GET, then you also need to:

- Install the GET license
- Create an additional storage bucket for the `admin` resources
- Disable the `gateway` used in open source Tempo
- Enable the `enterpriseGateway`, which is activated when you specify Enterprise

## Before you begin

These instructions are common across any flavor of Kubernetes. They also assume that you know how to install, configure, and operate a Kubernetes cluster.
It also assumes that you have an understanding of what the `kubectl` command does.

{{< admonition type="warning" >}}
This procedure is primarily aimed at local or development setups.
{{< /admonition >}}

### Hardware requirements

- Tempo: A single Kubernetes node with a minimum of 6 cores and 16 GB RAM
- GET: A single  Kubernetes node with a minimum of 9 cores and 32 GB RAM

### Software requirements

- Kubernetes 1.29 or later (refer to [Kubernetes installation documentation](https://kubernetes.io/docs/setup/))
- The `kubectl` command for your version of Kubernetes
- Helm 3 or later (refer to [Helm installation documentation](https://helm.sh/docs/intro/install/))
- GET only: [An enterprise license](https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/#obtain-a-get-license)

### Additional requirements

Verify that you have:

- Access to the Kubernetes cluster.
- Enabled persistent storage in the Kubernetes cluster, which has a default storage class setup.
- Access to a local storage option (like MinIO) or a storage bucket like Amazon S3, Azure Blob Storage, or Google Cloud Platform. Refer to the [Optional: Other storage options](#optional-other-storage-options) section for more information.
- DNS service works in the Kubernetes cluster. Refer to [Debugging DNS resolution](https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/) in the Kubernetes documentation.
- Optional: Set up an ingress controller in the Kubernetes cluster, for example [ingress-nginx](https://kubernetes.github.io/ingress-nginx/).

{{< admonition type="note" >}}
If you want to access Tempo from outside of the Kubernetes cluster, you may need an ingress.
Ingress-related procedures are optional.
{{< /admonition >}}

<!-- This section should be verified before being made visible. It's from Mimir and might need to be updated for Tempo.

## Security setup

This installation will not succeed if you have enabled the [PodSecurityPolicy](*https://v1-23.docs.kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#podsecuritypolicy) admission controller or if you are enforcing the Restricted policy with [Pod Security](https://v1-24.docs.kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-admission-labels-for-namespaces) admission controller. The reason is that the installation includes a deployment of MinIO. The [minio/minio chart](https://github.com/minio/minio/tree/master/helm/minio) is not compatible with running under a Restricted policy or the PodSecurityPolicy that the mimir-distributed chart provides.

If you are using the PodSecurityPolicy admission controller, then it is not possible to deploy the mimir-distributed chart with MinIO. Refer to Run Grafana Mimir in production using the Helm chart for instructions on setting up an external object storage and disable the built-in MinIO deployment with minio.enabled: false in the Helm values file.

If you are using the Pod Security admission controller, then MinIO and the tempo-distributed chart can successfully deploy under the baseline pod security level.
-->

## Create a custom namespace and add the Helm repository

Using a custom namespace solves problems later on because you don't have to overwrite the default namespace.

1. Create a unique Kubernetes namespace, for example `tempo-test`:

   ```bash
   kubectl create namespace tempo-test
   ```

  For more details, see the Kubernetes documentation about [Creating a namespace](https://kubernetes.io/docs/tasks/administer-cluster/namespaces/#creating-a-new-namespace).

1. Set up a Helm repository using the following commands:

   ```bash
   helm repo add grafana https://grafana.github.io/helm-charts
   helm repo update
   ```

   {{< admonition type="note" >}}
   The Helm chart at [https://grafana.github.io/helm-charts](https://grafana.github.io/helm-charts) is a publication of the source code at `grafana/tempo`.
   {{< /admonition >}}

## Set Helm chart values

The Helm chart for Tempo includes a file called `values.yaml`, which contains default configuration options.
In this procedure, you create a local file called `custom.yaml` in a working directory.

When you use Helm to deploy the chart, you can specify that Helm uses your `custom.yaml` to augment the default `values.yaml` file.
The `custom.yaml` file sets the storage and traces options, enables the gateway, and sets the cluster to main.
The traces section configures the distributor's receiver protocols.

After creating the file, you have the option to make changes in that file as needed for your deployment environment.

To customize your Helm chart values:

1. Create a `custom.yaml` file in your working directory.
1. From the examples below, copy and paste either the Tempo Helm chart values or the Grafana Enterprise Traces (GET) Helm chart values into your file.
1. Save your `custom.yaml` file.
1. For simple deployments, use the default `storage` and `minio` sections. The Helm chart deploys MinIO. Tempo uses it to store traces and other information, if you are running GET. Further down this page are instructions for customizing your trace storage configuration options.
1. Set your traces values to configure the receivers on the Tempo distributor.
1. Save the changes to your file.

### Tempo Helm chart values

This sample file contains example values for installing Tempo using Helm.

{{< collapse title="Example Tempo values file" >}}

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
# MinIO storage configuration
# Note: MinIO should not be used for production environments. This is for demonstration purposes only.
minio:
  enabled: true
  mode: standalone
  rootUser: grafana-tempo
  rootPassword: supersecret
  buckets:
    # Default Tempo storage bucket
    - name: tempo-traces
      policy: none
      purge: false
# Specifies which trace protocols to accept by the gateway.
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

{{< /collapse >}}

### Grafana Enterprise Traces helm chart values

The values in the example below provide configuration values for GET.
These values include an additional `admin` bucket and specifies a license.
The `enterpriseGateway` is automatically enabled as part of enabling the chart for installation of GET.

GET requires multitenancy. It must also be enabled explicitly in the values file.
For more information, refer to [Set up GET tenants](https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/set-up-get-tenants/).

{{< collapse title="GET example values file" >}}

```yaml
---
# Specify the global domain for the cluster (in this case just local cluster mDNS)
global:
  clusterDomain: 'cluster.local'

# Enable the Helm chart for GET installation
# Configure the Helm chart for a Grafana Enterprise Traces installation.
enterprise:
  enabled: true

# Enable multitenancy for GET (required)
multitenancyEnabled: true

# MinIO storage configuration
# The installs a separate MinIO service/deployment into the same cluster and namespace as the GET install.
# Note: MinIO should not be used for production environments.
minio:
  enabled: true
  mode: standalone
  rootUser: grafana-tempo
  rootPassword: supersecret
  buckets:
    # Bucket for traces storage if enterprise.enabled is true - requires license. This is where all trace span information is stored.
    - name: enterprise-traces
      policy: none
      purge: false
    # Admin client bucket if enterprise.enabled is true - requires license. This is where tenant and administration information is stored.
    - name: enterprise-traces-admin
      policy: none
      purge: false
  # Changed the mc (the MinIO CLI client) config path to '/tmp' from '/etc' as '/etc' is only writable by root and OpenShift will not permit this.
  configPathmc: '/tmp/minio/mc/'
storage:
  # Specifies traces storage location.
  # Uses the MinIO bucket configured for trace storage.
  trace:
    backend: s3
    s3:
      access_key: 'grafana-tempo'
      secret_key: 'supersecret'
      bucket: 'enterprise-traces'
      endpoint: 'tempo-minio:9000'
      insecure: true
  # Specifies administration data storage location.
  # Uses the MinIO bucket configured for admin storage.
  admin:
    backend: s3
    s3:
      access_key_id: 'grafana-tempo'
      secret_access_key: 'supersecret'
      bucket_name: 'enterprise-traces-admin'
      endpoint: 'tempo-minio:9000'
      insecure: true

# Specifies which trace protocols to accept by the gateway.
# Note: GET's Enterprise gateway will only accept OTLP over gRPC or HTTP.
traces:
  otlp:
    http:
      enabled: true
    grpc:
      enabled: true

# Configure the distributor component to log all received spans.
distributor:
  config:
    log_received_spans:
      enabled: true

# Specify the license. This is the base64 license text you have received from your Grafana Labs representative.
license:
  contents: |
    LICENSEGOESHERE
```

{{< /collapse >}}

#### Enterprise image version

If you require a different version of GET from the default in the Helm chart, update the `enterprise` configuration section in the `custom.yaml` values file with the required image version.
This example uses an image tag of v2.6.0.

```yaml
enterprise:
  enabled: true
  image:
    tag: v2.6.0
```

#### Enterprise license configuration

If you are using GET, you need to configure a license by either

- adding the license to the `custom.yaml` file or
- by using a secret that contains the license.

Only use one of these options.

{{< admonition type="note" >}}
The [Set up GET instructions](https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/#obtain-a-get-license) explain how to obtain a license.
{{< /admonition >}}

Using the first option, you can specify the license text in the `custom.yaml` values file created in the `license:` section.

```yaml
license:
  contents: |
    LICENSEGOESHERE
```

If you don't need to specify the license in the `custom.yaml` file, you can reference a secret that contains the license content.

1. Create the secret.

   ```bash
   kubectl -n tempo-test create secret generic tempo-license --from-file=license.jwt
   ```

1. Configure the `custom.yaml` that you created to reference the secret.

   ```yaml
   license:
     external: true
     secretName: get-license
   ```

### Set your storage option

Before you run the Helm chart, you need to configure where to store trace data.

The `storage` block defined in the `values.yaml` file configures the storage that Tempo uses for trace storage.

The procedure below configures MinIO as the local storage option managed by the Helm chart.
However, you can use another storage provider.
Refer to the Optional storage section.

{{< admonition type="note" >}}

The MinIO installation included with this Helm chart is for demonstration purposes only.
This configuration sets up a maximum storage size of 5GiB.
This MinIO installation isn't suitable for production environments and should only be used for example purposes.
For production, use performant, Enterprise-grade object storage.

{{< /admonition >}}

The Helm chart values provided include the basic MinIO set up values.
If you need to customize them, the steps below walk you through which sections to update.
If you don't need to change the values, you can skip this section.

1. Optional: Update the configuration options in `custom.yaml` for your configuration.

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

   Enterprise users also need to specify an additional bucket for `admin` resources.

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

1. Optional: If you need to change the defaults for MinIO, locate the MinIO section and change the relevant fields. The following example shows the username and password. Ensure that you update any `trace` or `admin` storage sections appropriately.

   ```yaml
   minio:
     enabled: true
     mode: standalone
     rootUser: minio
     rootPassword: minio123
   ```

#### Optional: Other storage options

You can enable persistent storage in the Kubernetes cluster, which has a default storage class setup.
To change the default, refer to the [StorageClass using Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/).

This Helm chart guide defaults to using MinIO as a simple solution to get you started.
However, you can use a storage bucket like Amazon S3, Azure Blob Storage, or Google Cloud Platform.

Each storage provider has a different configuration stanza.
You need to update your configuration based upon you storage provider.
Refer to the [`storage` configuration block](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#storage) for information on storage options.

Update the `storage` configuration options based upon your requirements:

- [Amazon S3 configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/s3). The Amazon S3 example is identical to the MinIO configuration, except the two last options, `endpoint` and `insecure`, are dropped.

- [Azure Blob Storage configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/azure)

- [Google Cloud Storage configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/gcs)

### Set traces receivers

The Helm chart values in your `custom.yaml` file are configured to use OTLP.
If you are using other receivers, then you need to configure them.

You can configure Tempo to receive data from OTLP, Jaeger, Zipkin, Kafka, and OpenCensus.
The following example enables OTLP on the distributor.
For other options, refer to the [distributor documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#distributor)

The example used in this procedure has OTLP enabled.

Enable any other protocols based on your requirements.

```yaml
traces:
  otlp:
    grpc:
      enabled: true
    http:
      enabled: true
```

For GET, the Enterprise Gateway is enabled by default, which only receives traces in OTLP gRPC and HTTP protocol.

### Optional: Add custom configurations

There are many configuration options available in the `tempo-distributed` Helm chart.
This procedure only covers the minimum configuration required to launch GET or Tempo in a basic deployment.

You can add values to your `custom.yaml` file to set custom configuration options that override the defaults present in the Helm chart.
The [`tempo-distributed` Helm chart's README](https://github.com/grafana/helm-charts/blob/main/charts/tempo-distributed/README.md) contains a list of available options.
The `values.yaml` files provides the defaults for the Helm chart.

Use the following command to see all of the configurable parameters for the `tempo-distributed` Helm chart:

```bash
helm show values grafana/tempo-distributed
```

Add the configuration sections to the `custom.yaml` file.
Include this file when you install or upgrade the Helm chart.

### Optional: Configure an ingress

An ingress lets you externally access a Kubernetes cluster.
Replace `<ingress-host>` with a suitable hostname that DNS can resolve to the external IP address of the Kubernetes cluster.
For more information, refer to [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

{{< admonition type="note" >}}
If you are using a Linux system and it's not possible for you set up local DNS resolution, use the `--add-host=<ingress-host>:<kubernetes-cluster-external-address>` command-line flag to define the `<ingress-host>` local address for the Docker commands in the examples that follow.
{{< /admonition >}}

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

### Optional: Configure TLS with Helm

Tempo and GET can be configured to communicate between the components using Transport Layer Security, or TLS.

To configure TLS with the Helm chart, you must have a TLS key-pair and CA certificate stored in a Kubernetes secret.

For instructions, refer to [Configure TLS with Helm](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/network/tls/).

### Optional: Use global or per-tenant overrides

The `tempo-distributed` Helm chart provides a module for users to set global or per-tenant override settings:

* Global overrides come under the `global_overrides` property, which pertain to the standard overrides
* Per-tenant overrides come under the `overrides` property, and allow specific tenants to alter configuration associated with them as per tenant-specific runtime overrides. The Helm chart generates a `/runtime/overrides.yaml` configuration file for all per-tenant configuration.

These overrides correlate to the standard (global) and tenant-specific (`per_tenant_overide_config`)overrides in Tempo and GET configuration.
For more information about overrides, refer to the [Overrides configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides) documentation.

Overrides can be used with both GET and Tempo.

The following example configuration sets some global configuration options, as well as a set of options for a specific tenant:

```yaml
global_overrides:
    defaults:
        ingestion:
          rate_limit_bytes: 5 * 1000 * 1000
          burst_size_bytes: 5 * 1000 * 1000
          max_traces_per_user: 1000
        global:
          max_bytes_per_trace: 10 * 1000 * 1000

        metrics_generator:
          processors: ['service-graphs', 'span-metrics']

overrides:
    '1234':
        ingestion:
          rate_limit_bytes: 2 * 1000 * 1000
          burst_size_bytes: 2 * 1000 * 1000
          max_traces_per_user: 400
        global:
          max_bytes_per_trace: 5 * 1000 * 1000
```

This configuration:

* Enables the Span Metrics and Service Graph metrics-generator processors for all tenants
* An ingestion rate and burst size limit of 5MB/s, a maximum trace size of 10MB and a maximum of 1000 live traces in an ingester for all tenants
* Overrides the '1234' tenant with a rate and burst size limit of 2MB/s, a maximum trace size of 5MB and a maximum of 400 live traces in an ingester

{{< admonition type="note" >}}
Runtime configurations should include all options for a specific tenant.
{{< /admonition >}}

## Install Grafana Tempo using the Helm chart

Use the following command to install Tempo using the configuration options you've specified in the `custom.yaml` file:

```bash
helm -n tempo-test install tempo grafana/tempo-distributed -f custom.yaml
```

{{< admonition type="note" >}}
The output of the command contains the write and read URLs necessary for the following steps.
{{< /admonition >}}

If the installation is successful, the output should be similar to this:

{{< collapse title="Installation block example" >}}

```bash
>  helm -n tempo-test install tempo grafana/tempo-distributed -f custom.yaml

W0210 15:02:09.901064    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.904082    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.906932    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.929946    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.930379    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
NAME: tempo
LAST DEPLOYED: Fri May 31 15:02:08 2024
NAMESPACE: tempo-test
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
***********************************************************************
 Welcome to Grafana Tempo
 Chart version: 1.10.1
 Tempo version: 2.6.1
***********************************************************************

Installed components:
* ingester
* distributor
* querier
* query-frontend
* compactor
* memcached
```
{{< /collapse >}}

{{< admonition type="note" >}}
If you update your `values.yaml` or `custom.yaml`, run the same helm install command and replace `install` with `upgrade`.
{{< /admonition >}}

Check the statuses of the Tempo pods:

```bash
kubectl -n tempo-test get pods
```

The results look similar to this:

```bash
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

Note that the `tempo-tokengen-job` has emitted a log message containing the initial `admin` token.

Retrieve the token with this command:

```bash
kubectl get pods | awk '/.*-tokengen-job-.*/ {print $1}' | xargs -I {} kubectl logs {} | awk '/Token:\s+/ {print $2}'
```

To get the logs for the `tokengen` Pod, you can use:

```bash
kubectl logs tempo-tokengen-job-58jhs
```

## Test your installation

The next step is to test your Tempo installation by sending trace data to Grafana.
You can use the [Set up a test application for a Tempo cluster](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/set-up-test-app) document for step-by-step instructions.

If you already have Grafana available, you can add a Tempo data source using the URL fitting to your environment.
For example:
`http://tempo-query-frontend.trace-test.svc.cluster.local:3100`

Enterprise users may need to [install the Enterprise Traces plugin](/docs/enterprise-traces/latest/setup/setup-get-plugin-grafana/) in their Grafana Enterprise instance to allow configuration of tenants, tokens, and access policies.
After creating a user and access policy using the plugin, you can configure a data source to point at `http://tempo-enterprise-gateway.tempo-test.svc.cluster.local:3100`.
