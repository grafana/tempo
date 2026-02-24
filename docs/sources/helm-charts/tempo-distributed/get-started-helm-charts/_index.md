---
description: Learn how to get started with Grafana Tempo using the Helm chart.
menuTitle: Get started
title: Get started with Grafana Tempo using the Helm chart
weight: 20
keywords:
  - Helm chart
  - Kubernetes
  - Grafana Tempo
topicType: task
versionDate: 2026-02-04
---

# Get started with Grafana Tempo using the Helm chart

{{< admonition type="note" >}}
The `tempo-distributed` Helm chart is now maintained by the community. 
The chart has moved to the [grafana-community/helm-charts](https://github.com/grafana-community/helm-charts/tree/main/charts/tempo-distributed) repository.
{{< /admonition >}}

The `tempo-distributed` Helm chart allows you to configure, install, and upgrade Grafana Tempo within a Kubernetes cluster.
Using this procedure, you need to:

- Create a custom namespace within your Kubernetes cluster
- Install Helm and the Grafana Community `helm-charts` repository
- Configure a storage option for traces
- Install Tempo using Helm

To learn more about Helm, read the [Helm documentation](https://helm.sh/).

## Before you begin

These instructions are common across any flavor of Kubernetes. They also assume that you know how to install, configure, and operate a Kubernetes cluster.
It also assumes that you have an understanding of what the `kubectl` command does.

{{< admonition type="warning" >}}
This procedure is primarily aimed at local or development setups.
{{< /admonition >}}

### Hardware requirements

- A single Kubernetes node with a minimum of 6 cores and 16 GB RAM

### Software requirements

- Kubernetes 1.29 or later (refer to [Kubernetes installation documentation](https://kubernetes.io/docs/setup/))
- The `kubectl` command for your version of Kubernetes
- Helm 3 or later (refer to [Helm installation documentation](https://helm.sh/docs/intro/install/))

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
{{< admonition type="note" >}}
If you want to access Tempo from outside of the Kubernetes cluster, you may need an ingress.
Ingress-related procedures are optional.

Note that [ingress-nginx](https://github.com/kubernetes/ingress-nginx) is being retired and should not be used in production environments.
{{< /admonition >}}
<!-- This section should be verified before being made visible. It's from Mimir and might need to be updated for Tempo.

## Security setup

This installation will not succeed if you have enabled the [PodSecurityPolicy](https://v1-23.docs.kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#podsecuritypolicy) admission controller or if you are enforcing the Restricted policy with [Pod Security](https://v1-24.docs.kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-admission-labels-for-namespaces) admission controller. The reason is that the installation includes a deployment of MinIO. The [minio/minio chart](https://github.com/minio/minio/tree/master/helm/minio) is not compatible with running under a Restricted policy or the PodSecurityPolicy that the mimir-distributed chart provides.

If you are using the PodSecurityPolicy admission controller, then it is not possible to deploy the mimir-distributed chart with MinIO. Refer to Run Grafana Mimir in production using the Helm chart for instructions on setting up an external object storage and disable the built-in MinIO deployment with minio.enabled: false in the Helm values file.

If you are using the Pod Security admission controller, then MinIO and the tempo-distributed chart can successfully deploy under the baseline pod security level.
-->

## Create a custom namespace and add the Helm repository

Using a custom namespace solves problems later on because you don't have to overwrite the default namespace.

1. Create a unique Kubernetes namespace, for example `tempo-test`:

   ```bash
   kubectl create namespace tempo-test
   ```

For more details, refer to the Kubernetes documentation about [Creating a namespace](https://kubernetes.io/docs/tasks/administer-cluster/namespaces/#creating-a-new-namespace).

1. Set up a Helm repository using the following commands:

   ```bash
   helm repo add grafana-community https://grafana-community.github.io/helm-charts
   helm repo update
   ```

   {{< admonition type="note" >}}
   The Helm chart at [https://grafana-community.github.io/helm-charts](https://grafana-community.github.io/helm-charts) is maintained by the community. The chart source code is available at [grafana-community/helm-charts](https://github.com/grafana-community/helm-charts).
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
1. From the example below, copy and paste the Tempo Helm chart values into your file.
1. Save your `custom.yaml` file.
1. For simple deployments, use the default `storage` and `minio` sections. The Helm chart deploys MinIO. Tempo uses it to store traces. Further down this page are instructions for customizing your trace storage configuration options.
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
      access_key: "grafana-tempo"
      secret_key: "supersecret"
      bucket: "tempo-traces"
      endpoint: "tempo-minio:9000"
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
 ```

{{< /collapse >}}

### Set your storage option

Before you run the Helm chart, you need to configure where to store trace data.

The `storage` block defined in the `values.yaml` file configures the storage that Tempo uses for trace storage.

The procedure below configures MinIO as the local storage option managed by the Helm chart.
However, you can use another storage provider.
Refer to the [Optional: Other storage options](#optional-other-storage-options) section.

{{< admonition type="note" >}}

The MinIO installation included with this Helm chart is for demonstration purposes only. MinIO is deprecated and in maintenance mode.
This configuration sets up a maximum storage size of 5GiB.
This MinIO installation isn't suitable for production environments and should only be used for example purposes.
For production, use performant, production-grade object storage.

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
         access_key: "grafana-tempo"
         secret_key: "supersecret"
         bucket: "tempo-traces"
         endpoint: "tempo-minio:9000"
         insecure: true
   ```

1. Optional: If you need to change the defaults for MinIO, locate the MinIO section and change the relevant fields. The following example shows the username and password. Ensure that you update any `trace` storage sections appropriately.

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

#### Azure with the `local_blocks` and metrics-generator processors

[//]: # "Shared content for local_blocks and metrics-generator in Azure blob storage when using Helm"
[//]: # "This content is located in /tempo/docs/sources/shared/azure-metrics-generator.md"

{{< docs/shared source="tempo" lookup="azure-metrics-generator.md" version="<TEMPO_VERSION>" >}}

For more information about the local-blocks processor, refer to [Configure TraceQL metrics](https://grafana.com/docs/tempo/next/metrics-from-traces/metrics-queries/configure-traceql-metrics).

### Set traces receivers

The Helm chart values in your `custom.yaml` file are configured to use OTLP.
If you are using other receivers, then you need to configure them.

You can configure Tempo to receive data from OTLP, Jaeger, Zipkin and Kafka.
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

### Optional: Add custom configurations

There are many configuration options available in the `tempo-distributed` Helm chart.
This procedure only covers the minimum configuration required to launch Tempo in a basic deployment.

You can add values to your `custom.yaml` file to set custom configuration options that override the defaults present in the Helm chart.
The [`tempo-distributed` Helm chart's README](https://github.com/grafana-community/helm-charts/blob/main/charts/tempo-distributed/README.md) contains a list of available options.
The `values.yaml` files provides the defaults for the Helm chart.

Use the following command to see all of the configurable parameters for the `tempo-distributed` Helm chart:

```bash
helm show values grafana-community/tempo-distributed
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

Tempo can be configured to communicate between the components using Transport Layer Security, or TLS.

To configure TLS with the Helm chart, you must have a TLS key-pair and CA certificate stored in a Kubernetes secret.

For instructions, refer to [Configure TLS with Helm](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/network/tls/).

### Optional: Use global or per-tenant overrides

The `tempo-distributed` Helm chart provides a module for users to set global or per-tenant override settings:

- Global overrides come under the `overrides` property, which pertain to the standard overrides
- Per-tenant overrides come under the `per_tenant_overrides` property, and allow specific tenants to alter configuration associated with them as per tenant-specific runtime overrides. The Helm chart generates a `/runtime/overrides.yaml` configuration file for all per-tenant configuration.

These overrides correlate to the standard (global) and tenant-specific (`per_tenant_overide_config`)overrides in Tempo configuration.
For more information about overrides, refer to the [Overrides configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides) documentation.

The following example configuration sets some global configuration options, as well as a set of options for a specific tenant:

```yaml
overrides:
  defaults:
    ingestion:
      rate_limit_bytes: 5 * 1000 * 1000
      burst_size_bytes: 5 * 1000 * 1000
      max_traces_per_user: 1000
    global:
      max_bytes_per_trace: 10 * 1000 * 1000

    metrics_generator:
      processors: ["service-graphs", "span-metrics"]

per_tenant_overrides:
  "1234":
    ingestion:
      rate_limit_bytes: 2 * 1000 * 1000
      burst_size_bytes: 2 * 1000 * 1000
      max_traces_per_user: 400
    global:
      max_bytes_per_trace: 5 * 1000 * 1000
```

This configuration:

- Enables the Span Metrics and Service Graph metrics-generator processors for all tenants
- An ingestion rate and burst size limit of 5MB/s, a maximum trace size of 10MB and a maximum of 1000 live traces in an ingester for all tenants
- Overrides the '1234' tenant with a rate and burst size limit of 2MB/s, a maximum trace size of 5MB and a maximum of 400 live traces in an ingester

{{< admonition type="note" >}}
Runtime configurations should include all options for a specific tenant.
{{< /admonition >}}

## Install Grafana Tempo using the Helm chart

Use the following command to install Tempo using the configuration options you've specified in the `custom.yaml` file:

```bash
helm -n tempo-test install tempo grafana-community/tempo-distributed -f custom.yaml
```

{{< admonition type="note" >}}
The output of the command contains the write and read URLs necessary for the following steps.
{{< /admonition >}}

If the installation is successful, the output should be similar to this:

{{< collapse title="Installation block example" >}}

```bash
>  helm -n tempo-test install tempo grafana-community/tempo-distributed -f custom.yaml

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

## Test your installation

The next step is to test your Tempo installation by sending trace data to Grafana.
You can use the [Set up a test application for a Tempo cluster](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/set-up-test-app) document for step-by-step instructions.

If you already have Grafana available, you can add a Tempo data source using the URL fitting to your environment, for example:
`http://tempo-query-frontend.trace-test.svc.cluster.local:3100`

## Set up metamonitoring

Metamonitoring provides observability for your Tempo deployment by collecting metrics and logs from the Tempo components themselves. This helps you monitor the health and performance of your tracing infrastructure.
Setting up metamonitoring for Tempo uses the `k8s-monitoring` Helm chart.
For more information about this Helm chart, refer to [k8s-monitoring README](https://github.com/grafana/k8s-monitoring-helm/blob/main/charts/k8s-monitoring/README.md).

### Configure metamonitoring

To configure metamonitoring, you need to create a `metamonitoring-values.yaml` file and use the Kubernetes Monitoring Helm chart.

1. Create a `metamonitoring-values.yaml` file for the Kubernetes Monitoring Helm chart configuration.

   Replace the following values with your monitoring backend details:

   - `tempo`: A descriptive name for your cluster and namespace
   - `<url>`: Your Prometheus and Loki endpoint URLs
   - `<username>`: Your username/instance ID
   - `<password>`: Your password/API key

   ```yaml
   cluster:
     name: tempo # Name of the cluster, this will populate the cluster label

   integrations:
     tempo:
       instances:
         - name: "tempo" # This is the name for the instance label that will be reported.
           namespaces:
             - tempo # This is the namespace that will be searched for tempo instances, change this accordingly
           metrics:
             enabled: true
             portName: prom-metrics
           logs:
             enabled: true
           labelSelectors:
             app.kubernetes.io/name: tempo

     alloy:
       name: "alloy-tempo"

   destinations:
     - name: "tempo-metrics"
       type: prometheus
       url: "<url>" # Enter Prometheus URL
       auth:
         type: basic
         username: "<username>" # Enter username
         password: "<password>" # Enter password

     - name: "tempo-logs"
       type: loki
       url: "<url>" # Enter Loki URL
       auth:
         type: basic
         username: "<username>" # Enter username
         password: "<password>" # Enter password

   alloy-metrics:
     enabled: true

   podLogs:
     enabled: true
     gatherMethod: kubernetesApi
     namespaces: [tempo] # Set to namespace
     collector: alloy-singleton

   alloy-singleton:
     enabled: true

   alloy-metrics:
     enabled: true # This will send Grafana Alloy metrics to ensure the monitoring is working properly.
   ```

1. Install the k8s-monitoring Helm chart:

   ```bash
   helm install k8s-monitoring grafana/k8s-monitoring \
     --namespace monitoring \
     --create-namespace \
     -f metamonitoring-values.yaml
   ```

1. Verify the installation:

   ```bash
   kubectl -n monitoring get pods
   ```

   You should see pods for the k8s-monitoring components running.

### Verify metamonitoring in Grafana

Verify that metamonitoring is working correctly by checking metrics and logs in Grafana.

1. Navigate to your Grafana instance (Grafana Cloud or self-hosted).

1. Check that metrics are being collected:

   - Go to **Explore** > **Prometheus**.
   - Query for Tempo metrics like `tempo_build_info` or `tempo_distributor_spans_received_total`.

1. Check that logs are being collected:

   - Go to **Explore** > **Loki**
   - Filter logs by your cluster name and look for Tempo component logs

1. Set up Tempo monitoring dashboards:
   - For pre-built dashboards and alerts, refer to the [Tempo mixin documentation](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin)

Your Tempo deployment now includes comprehensive metamonitoring, giving you visibility into the health and performance of your tracing infrastructure.
