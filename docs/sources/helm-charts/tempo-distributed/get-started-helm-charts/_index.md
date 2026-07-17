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
versionDate: 2026-07-15
---

# Get started with Grafana Tempo using the Helm chart

{{< admonition type="note" >}}
The `tempo-distributed` Helm chart is now maintained by the community.
The chart has moved to the [grafana-community/helm-charts](https://github.com/grafana-community/helm-charts/tree/main/charts/tempo-distributed) repository.
{{< /admonition >}}

{{< admonition type="note" >}}
Grafana Enterprise Traces (GET) is no longer maintained in this chart. The enterprise templates (`admin-api`, `enterprise-gateway`, `enterprise-federation-frontend`, `provisioner`, and `tokengen`) have been removed. For enterprise deployments, use the [grafana/helm-charts](https://github.com/grafana/helm-charts) repository.
{{< /admonition >}}

The `tempo-distributed` Helm chart allows you to configure, install, and upgrade Grafana Tempo within a Kubernetes cluster.
Tempo 3.0 uses a Kafka-based architecture: distributors write spans to a Kafka-compatible broker, block-builders consume from Kafka to write blocks to object storage, and live-stores serve recent-data queries.
As a result, this procedure requires an external Kafka-compatible broker and an external S3-compatible object store, neither of which the chart deploys for you.

Using this procedure, you need to:

- Create a custom namespace within your Kubernetes cluster
- Install Helm and the Grafana Community `helm-charts` repository
- Provide an external S3-compatible object store for traces
- Provide a Kafka-compatible broker for the ingest path
- Install Tempo using Helm

To learn more about Helm, read the [Helm documentation](https://helm.sh/).

## Before you begin

These instructions are common across any flavor of Kubernetes. They also assume that you know how to install, configure, and operate a Kubernetes cluster.
It also assumes that you have an understanding of what the `kubectl` command does.

{{< admonition type="warning" >}}
This procedure is primarily aimed at local or development setups.
{{< /admonition >}}

### Hardware requirements

The Tempo 3.0 microservices deployment runs more components than earlier versions, including block-builders, live-stores, and a backend-scheduler and backend-worker, alongside an external Kafka broker and object store.

- The main example uses the chart default of three Kafka partitions, which runs three block-builder and three live-store replicas. For a smaller single-partition footprint suitable for a single node (one block-builder and one live-store replica), use the [Optional: Quick test with a local S3-compatible store and Kafka](#optional-quick-test-with-a-local-s3-compatible-store-and-kafka) section and a Kubernetes node with a minimum of 6 cores and 16 GB RAM.
- Production deployments scale block-builders and live-stores with your Kafka partition count and require significantly more resources. Refer to [Plan your deployment](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/) for sizing guidance.

### Software requirements

- Kubernetes 1.25 or later (refer to [Kubernetes installation documentation](https://kubernetes.io/docs/setup/))
- The `kubectl` command for your version of Kubernetes
- Helm 3 or later (refer to [Helm installation documentation](https://helm.sh/docs/intro/install/))
- A Kafka-compatible broker reachable from the cluster (for example, Apache Kafka, Redpanda, or WarpStream). The chart doesn't deploy Kafka.

### Additional requirements

Verify that you have:

- Access to the Kubernetes cluster.
- Enabled persistent storage in the Kubernetes cluster, which has a default storage class setup.
- Access to an external S3-compatible object store, such as Amazon S3, Azure Blob Storage, or Google Cloud Platform. The chart no longer bundles MinIO. For local testing, you can run an S3-compatible store yourself using one of the options in [S3-compatible local stores for testing](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/s3/#s3-compatible-local-stores-for-testing). Refer to the [Optional: Other storage options](#optional-other-storage-options) section for more information.
- A reachable Kafka-compatible broker for the ingest path.
- DNS service works in the Kubernetes cluster. Refer to [Debugging DNS resolution](https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/) in the Kubernetes documentation.
- Optional: Set up an ingress controller in the Kubernetes cluster, for example [ingress-nginx](https://kubernetes.github.io/ingress-nginx/).

{{< admonition type="note" >}}
If you want to access Tempo from outside of the Kubernetes cluster, you may need an ingress.
Ingress-related procedures are optional.

Note that [ingress-nginx](https://github.com/kubernetes/ingress-nginx) is being retired and shouldn't be used in production environments.
{{< /admonition >}}

## Create a custom namespace and add the Helm repository

Using a custom namespace solves problems later on because you don't have to overwrite the default namespace.

1. Create a unique Kubernetes namespace, for example, `tempo-test`:

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
   The Helm chart at [https://grafana-community.github.io/helm-charts](https://grafana-community.github.io/helm-charts) is maintained by the community.
   {{< /admonition >}}

## Set Helm chart values

The Helm chart for Tempo includes a file called `values.yaml`, which contains default configuration options.
In this procedure, you create a local file called `custom.yaml` in a working directory.

When you use Helm to deploy the chart, you can specify that Helm uses your `custom.yaml` to augment the default `values.yaml` file.
The `custom.yaml` file sets the storage, ingest, and traces options.
The `storage` section points Tempo at your object store, the `ingest` section connects Tempo to your Kafka broker, and the `traces` section configures the distributor's receiver protocols.

After creating the file, you have the option to make changes in that file as needed for your deployment environment.

To customize your Helm chart values:

1. Create a `custom.yaml` file in your working directory.
1. From the example below, copy and paste the Tempo Helm chart values into your file.
1. Save your `custom.yaml` file.
1. Set the `storage` section to point at your external S3-compatible object store. Further down this page are instructions for customizing your trace storage configuration options.
1. Set the `ingest.kafka` section to point at your Kafka broker, and set `blockBuilder.replicas` and `liveStore.replicas` to match your Kafka partition count.
1. Set your traces values to configure the receivers on the Tempo distributor.
1. Save the changes to your file.

### Tempo Helm chart values

This sample file contains example values for installing Tempo using Helm.

{{< collapse title="Example Tempo values file" >}}

```yaml
---
# Point Tempo at your external S3-compatible object store.
storage:
  trace:
    backend: s3
    s3:
      access_key: "grafana-tempo"
      secret_key: "supersecret"
      bucket: "tempo-traces"
      endpoint: "<s3-endpoint>:9000"
      insecure: true # remove once TLS is configured
# Connect Tempo to your Kafka-compatible broker.
ingest:
  kafka:
    address: "<kafka-broker>:9092"
    topic: tempo-traces
    auto_create_topic_enabled: true
    # Number of partitions for the auto-created topic. This example uses the chart
    # default. The block-builder and live-store replica counts must equal this value.
    # Tune this value up for production based on your throughput requirements.
    auto_create_topic_default_partitions: 3
# One block-builder replica per Kafka partition.
blockBuilder:
  replicas: 3
# One live-store replica per Kafka partition.
liveStore:
  replicas: 3
# Specifies which trace protocols the distributor accepts.
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

{{< admonition type="warning" >}}
The chart no longer bundles MinIO. Setting `minio.enabled: true` now fails the install.
You must point Tempo at an externally managed S3-compatible object store through the `storage.trace.s3` block.
{{< /admonition >}}

Configure the `storage.trace.s3` block in your `custom.yaml` to point at your object store:

```yaml
---
storage:
  trace:
    backend: s3
    s3:
      access_key: "grafana-tempo"
      secret_key: "supersecret"
      bucket: "tempo-traces"
      endpoint: "<s3-endpoint>:9000"
      insecure: true # remove once TLS is configured
```

The Amazon S3 example is identical, except the `endpoint` and `insecure` options are dropped.
For other providers, refer to the [Optional: Other storage options](#optional-other-storage-options) section.

#### Optional: Quick test with a local S3-compatible store and Kafka

For local testing only, you can run an S3-compatible object store and a Kafka broker yourself, then point Tempo at them.
Neither is suitable for production.

1. Stand up a local S3-compatible object store using one of the options in [S3-compatible local stores for testing](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/s3/#s3-compatible-local-stores-for-testing). SeaweedFS is the recommended option. Create a bucket named `tempo-traces`, then set `storage.trace.s3.endpoint`, `access_key`, and `secret_key` in your `custom.yaml` to match.

1. Run a single-node Kafka broker in the cluster. The following manifest deploys a plain Kafka broker in a `kafka-test` namespace, suitable for evaluation:

   ```yaml
   apiVersion: v1
   kind: Namespace
   metadata:
     name: kafka-test
   ---
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: kafka
     namespace: kafka-test
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: kafka
     template:
       metadata:
         labels:
           app: kafka
       spec:
         containers:
           - name: kafka
             image: apache/kafka:4.0.0
             ports:
               - containerPort: 9092
                 name: plaintext
             env:
               - name: KAFKA_NODE_ID
                 value: "1"
               - name: KAFKA_PROCESS_ROLES
                 value: "broker,controller"
               - name: KAFKA_LISTENERS
                 value: "PLAINTEXT://:9092,CONTROLLER://:9093"
               - name: KAFKA_ADVERTISED_LISTENERS
                 value: "PLAINTEXT://kafka.kafka-test.svc.cluster.local:9092"
               - name: KAFKA_CONTROLLER_LISTENER_NAMES
                 value: "CONTROLLER"
               - name: KAFKA_LISTENER_SECURITY_PROTOCOL_MAP
                 value: "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT"
               - name: KAFKA_CONTROLLER_QUORUM_VOTERS
                 value: "1@localhost:9093"
               - name: KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR
                 value: "1"
               - name: KAFKA_AUTO_CREATE_TOPICS_ENABLE
                 value: "true"
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: kafka
     namespace: kafka-test
   spec:
     selector:
       app: kafka
     ports:
       - name: plaintext
         port: 9092
         targetPort: 9092
   ```

1. Set `ingest.kafka.address` in your `custom.yaml` to `kafka.kafka-test.svc.cluster.local:9092`. For this single-node local test, set `blockBuilder.replicas`, `liveStore.replicas`, and `ingest.kafka.auto_create_topic_default_partitions` all to `1` to reduce the resource footprint. For production, use the chart default of `3` partitions or higher, and keep the block-builder and live-store replica counts equal to the partition count.

#### Optional: Other storage options

You can enable persistent storage in the Kubernetes cluster, which has a default storage class setup.
To change the default, refer to the [StorageClass using Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/).

You can use any S3-compatible store, or a bucket like Amazon S3, Azure Blob Storage, or Google Cloud Platform.

Each storage provider has a different configuration stanza.
You need to update your configuration based upon your storage provider.
Refer to the [`storage` configuration block](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#storage) for information on storage options.

Update the `storage` configuration options based upon your requirements:

- [Amazon S3 configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/s3). The Amazon S3 example is identical to the S3-compatible example above, except the `endpoint` and `insecure` options are dropped.

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
   gateway:
     enabled: true
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
- An ingestion rate and burst size limit of 5MB/s, a maximum trace size of 10MB and a maximum of 1000 live traces per tenant for all tenants
- Overrides the '1234' tenant with a rate and burst size limit of 2MB/s, a maximum trace size of 5MB and a maximum of 400 live traces per tenant

{{< admonition type="note" >}}
Runtime configurations should include all options for a specific tenant.
{{< /admonition >}}

{{< admonition type="note" >}}
The `local-blocks` processor was removed in Tempo 3.0. Block building is handled by the block-builder component, so you can no longer enable `local-blocks` through `metrics_generator.processors`.
{{< /admonition >}}

## Install Tempo using the Helm chart

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

NAME: tempo
LAST DEPLOYED: Fri May 31 15:02:08 2026
NAMESPACE: tempo-test
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
***********************************************************************
 Welcome to Grafana Tempo
 Chart version: 3.0.5
 Tempo version: 3.0.2
***********************************************************************

Installed components:
* distributor
* querier
* query-frontend
* backend-scheduler
* backend-worker
* block-builder
* live-store
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
tempo-backend-scheduler-0               1/1     Running   0          22h
tempo-backend-worker-0                  1/1     Running   0          22h
tempo-block-builder-0                   1/1     Running   0          22h
tempo-block-builder-1                   1/1     Running   0          22h
tempo-block-builder-2                   1/1     Running   0          22h
tempo-distributor-bbf4889db-v8l8r       1/1     Running   0          22h
tempo-live-store-0                      1/1     Running   0          22h
tempo-live-store-1                      1/1     Running   0          22h
tempo-live-store-2                      1/1     Running   0          22h
tempo-memcached-0                       1/1     Running   0          8d
tempo-querier-777c8dcf54-fqz45          1/1     Running   0          22h
tempo-query-frontend-7f7f686d55-xsnq5   1/1     Running   0          22h
```

Wait until all of the pods have a status of Running or Completed, which might take a few minutes.

## Test your installation

The next step is to test your Tempo installation by sending trace data to Grafana.
You can use the [Set up a test application for a Tempo cluster](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/test/set-up-test-app/) document for step-by-step instructions.

If you already have Grafana available, you can add a Tempo data source using the URL fitting to your environment, for example:
`http://tempo-query-frontend.tempo-test.svc.cluster.local:3200`

{{< admonition type="note" >}}
If you enable the gateway (`gateway.enabled: true`), point the data source at the gateway service instead of the query-frontend.
{{< /admonition >}}

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

1. Navigate to your Grafana instance (Grafana Cloud or self-managed).

1. Check that metrics are being collected:

   - Go to **Explore** > **Prometheus**.
   - Query for Tempo metrics like `tempo_build_info` or `tempo_distributor_spans_received_total`.

1. Check that logs are being collected:

   - Go to **Explore** > **Loki**
   - Filter logs by your cluster name and look for Tempo component logs

1. Set up Tempo monitoring dashboards:
   - For pre-built dashboards and alerts, refer to the [Tempo mixin documentation](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin)

Your Tempo deployment now includes comprehensive metamonitoring, giving you visibility into the health and performance of your tracing infrastructure.
