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

The Grafana Tempo Helm chart allows you to configure, install, and upgrade Grafana Tempo within a Kubernetes cluster. Using this procedure, you will:

- Create a custom namespace within your Kubernetes cluster
- Install Helm and the Grafana `helm-charts` repository
- Configure a storage option for traces
- Install Tempo using Helm

To learn more about Helm, read the [Helm documentation](https://helm.sh/).

If you are using Helm to install Grafana Enterprise Traces (GET), then you also need to:

- Install the GET license
- Create an additional storage bucket for the `admin` resources
- Disable the `gateway`
- Enable the `enterpriseGateway`

## Before you begin

These instructions are common across any flavor of Kubernetes. They also assume that you know how to install, configure, and operate a Kubernetes cluster.
It also assumes that you have an understanding of what the `kubectl` command does.

{{< admonition type="warning" >}}
This procedure is primarily aimed at local or development setups.
{{< /admonition >}}

### Hardware requirements

- A single Kubernetes node with a minimum of 4 cores and 16 GB RAM

### Software requirements

- Kubernetes 1.20 or later (refer to [Kubernetes installation documentation](https://kubernetes.io/docs/setup/))
- The `kubectl` command for your version of Kubernetes
- Helm 3 or later (refer to [Helm installation documentation](https://helm.sh/docs/intro/install/))
- GET only: [An enterprise license](/docs/enterprise-traces/latest/setup/#obtain-a-get-license)

### Additional requirements

Verify that you have:

- Access to the Kubernetes cluster
- Enabled persistent storage in the Kubernetes cluster, which has a default storage class setup.
- Access to a local storage option (like MinIO) or a storage bucket like Amazon S3, Azure Blob Storage, or Google Cloud Platform (refer to [Optional: Other storage options](#optional-other-storage-options) section for more information)
- DNS service works in the Kubernetes cluster (refer to [Debugging DNS resolution](https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/) in the Kubernetes documentation)
- Optional: Set up an ingress controller in the Kubernetes cluster, for example [ingress-nginx](https://kubernetes.github.io/ingress-nginx/)

{{< admonition type="note" >}}
If you want to access Tempo from outside of the Kubernetes cluster, you may need an ingress.
Ingress-related procedures are optional.
{{< /admonition >}}

<!-- This section should be verified before being made visible. It’s from Mimir and might need to be updated for Tempo.

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

   {{< admonition type="note" >}} The Helm chart at https://grafana.github.io/helm-charts is a publication of the source code at `grafana/tempo`.
   {{< /admonition >}}

## Set Helm chart values

The Helm Chart for Tempo includes a file called "values.yaml", containing default configuration options.
In this case, you create a local file called `custom.yaml` in a working directory.

After creating the file, you have the option to make changes in that file as needed for your deployment environment.

When you use Helm to deploy the chart, you can specify that Helm uses your `custom.yaml` instead of `values.yaml`.
The `custom.yaml` file sets the storage and traces options, enables the gateway, and sets the cluster to main.
The `traces` configure the distributor's receiver protocols.

To customize your Helm chart values:

1. Create a `custom.yaml` file in your working directory.
1. From the examples below, copy and paste either the Tempo Helm chart values or the Grafana Enterprise Traces (GET) Helm chart values into your file.
1. Save your `custom.yaml` file.
1. For simple deployments, use the default `storage` and `minio` sections. The Helm chart deploys MinIO. Tempo uses it to store traces and other information, if you are running GET. Further down this page are instructions for customizing your trace storage configuration options.
1. Set your traces values to configure the receivers on the Tempo distributor.
1. Save the changes to your file.

### Tempo Helm chart values

This sample file contains example values for installing Tempo using Helm.

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
#MinIO storage configuration
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
These values include an additional `admin` bucket, disables the `gateway`, enables the `enterpriseGateway`, and specifies a license.

```yaml
---
global:
  clusterDomain: 'cluster.local'

multitenancyEnabled: true
enterprise:
  enabled: true
  image:
    tag: v2.4.1
enterpriseGateway:
  enabled: true
gateway:
  enabled: false
# MinIO storage configuration
minio:
  enabled: true
  mode: standalone
  rootUser: grafana-tempo
  rootPassword: supersecret
  buckets:
    # Bucket for traces storage if enterprise.enabled is true - requires license.
    - name: enterprise-traces
      policy: none
      purge: false
    # Admin client bucket if enterprise.enabled is true - requires license.
    - name: enterprise-traces-admin
      policy: none
      purge: false
  # Changed the mc config path to '/tmp' from '/etc' as '/etc' is only writable by root and OpenShift will not permit this.
  configPathmc: '/tmp/minio/mc/'
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
Only use one of these options.

{{< admonition type="note" >}}
The [Set up GET instructions](/docs/enterprise-traces/latest/setup/#obtain-a-get-license) explain how to obtain a license.
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
   ```

### Set your storage option

Before you run the Helm chart, you need to configure where to store trace data.

The `storage` block defined in the `values.yaml` file configures the storage that Tempo uses for trace storage.

The procedure below configures MinIO as the local storage option managed by the Helm chart.
However, you can use a other storage provides. Refer to the Optional storage section below.

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

   Enterprise users may also need to specify an additional bucket for `admin` resources.

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

1. Optional: Locate the MinIO section and change the username and password.

   ```yaml
   minio:
     enabled: true
     mode: standalone
     rootUser: minio
     rootPassword: minio123
   ```

### Optional: Other storage options

Persistent storage is enabled in the Kubernetes cluster, which has a default storage class setup.
You can change the default [StorageClass using Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/).

This Helm chart guide defaults to using MinIO as a simple solution to get you started.
However, you can use a storage bucket like Amazon S3, Azure Blob Storage, or Google Cloud Platform.

Each storage provider has a different configuration stanza.
You need to update your configuration based upon you storage provider.
Refer to the [`storage` configuration block]({{< relref "/docs/tempo/latest/configuration#storage" >}}) for information on storage options.

To use other storage options, set `minio.enabled: false` in the `values.yaml` file:

```yaml
---
minio:
  enabled: false # Disables the MinIO chart
```

Update the `storage` configuration options based upon your requirements:

- [Amazon S3 configuration documentation]({{< relref "/docs/tempo/latest/configuration/hosted-storage/s3" >}}). The Amazon S3 example is identical to the MinIO configuration, except the two last options, `endpoint` and `insecure`, are dropped.

- [Azure Blob Storage configuration documentation]({{< relref "/docs/tempo/latest/configuration/hosted-storage/azure" >}})

- [Google Cloud Storage configuration documentation]({{< relref "/docs/tempo/latest/configuration/hosted-storage/gcs" >}})

### Set traces receivers

The Helm chart values in your `custom.yaml` file are configure to use OTLP.
If you are using other receivers, then you need to configure them.

You can configure Tempo can to receive data from OTLP, Jaegar, Zipkin, Kafka, and OpenCensus.
The following example enables OTLP on the distributor. For other options, refer to the [distributor documentation]({{< relref "/docs/tempo/latest/configuration#distributor" >}})

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

There are many configuration options available in the `tempo-distributed` Helm chart.
This procedure only covers the bare minimum required to launch GET or Tempo in a basic deployment.

You can add values to your `custom.yaml` file to set custom configuration options that override the defaults present in the Helm chart.
The [`tempo-distributed` Helm chart's README](https://github.com/grafana/helm-charts/blob/main/charts/tempo-distributed/README.md) contains a list of available options.
The `values.yaml` files provides the defaults for the Helm chart.

Use the following command to see all of the configurable parameters for the `tempo-distributed` Helm chart:

```bash
helm show values grafana/tempo-distributed
```

Add the configuration sections to the `custom.yaml` file.
Include this file when you install or upgrade the Helm chart.

#### Optional: Configure an ingress

An ingress lets you externally access a Kubernetes cluster.
Replace `<ingress-host>` with a suitable hostname that DNS can resolve to the external IP address of the Kubernetes cluster.
For more information, refer to [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

{{< admonition type="note" >}}
On Linux systems, and if it's not possible for you set up local DNS resolution, you can use the `--add-host=<ingress-host>:<kubernetes-cluster-external-address>` command-line flag to define the `<ingress-host>` local address for the docker commands in the examples that follow.
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

## Install Grafana Tempo using the Helm chart

Use the following command to install Tempo using the configuration options you’ve specified in the `custom.yaml` file:

```bash
helm -n tempo-test install tempo grafana/tempo-distributed -f custom.yaml
```

{{< admonition type="note" >}}
The output of the command contains the write and read URLs necessary for the following steps.
{{< /admonition >}}

If the installation is successful, the output should be similar to this:

```bash
>  helm -n tempo-test install tempo grafana/tempo-distributed -f custom.yaml

W0210 15:02:09.901064    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.904082    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.906932    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.929946    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
W0210 15:02:09.930379    8613 warnings.go:70] spec.template.spec.topologySpreadConstraints[0].topologyKey: failure-domain.beta.kubernetes.io/zone is deprecated since v1.17; use "topology.kubernetes.io/zone" instead
NAME: tempo
LAST DEPLOYED: Fri May 31 15:02:08 2023
NAMESPACE: tempo-test
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
***********************************************************************
 Welcome to Grafana Tempo
 Chart version: 1.10.1
 Tempo version: 2.5.0
***********************************************************************

Installed components:
* ingester
* distributor
* querier
* query-frontend
* compactor
* memcached
```

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
You can use the [Set up a test application for a Tempo cluster]({{< relref "/docs/tempo/latest/setup/set-up-test-app" >}}) document for step-by-step instructions.

If you already have Grafana available, you can add a Tempo data source using the URL fitting to your environment.
For example:
`http://tempo-query-frontend.trace-test.svc.cluster.local:3100`

Enterprise users may need to [install the Enterprise Traces plugin](/docs/enterprise-traces/latest/setup/setup-get-plugin-grafana/) in their Grafana Enterprise instance to allow configuration of tenants, tokens, and access policies.
After creating a user and access policy using the plugin, you can configure a data source to point at `http://tempo-enterprise-gateway.tempo-test.svc.cluster.local:3100`.
