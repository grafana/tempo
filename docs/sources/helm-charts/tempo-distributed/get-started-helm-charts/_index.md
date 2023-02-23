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

- Create a custom name-space within your Kubernetes cluster
- Install Helm and the grafana helm-charts repository
- Configure Google Cloud Platform storage for traces
- Install Tempo using Helm

To learn more about Helm, read the [Helm documentation](https://helm.sh/).

## Before you begin

These instructions are common across any flavor of Kubernetes. They also assume that you know how to install, configure, and operate a Kubernetes cluster.
It also assumes that you have an understanding of what the `kubectl` command does.

>**CAUTION**: This procedure is primarily aimed at local or development setups.

### Hardware requirements

- A single Kubernetes node with a minimum of 4 cores and 16GiB RAM

### Software requirements

- Kubernetes 1.20 or later (see [Kubernetes installation documentation](https://kubernetes.io/docs/setup/))
- The `kubectl` command for your version of Kubernetes
- Helm 3 or later (see [Helm installation documentation](https://helm.sh/docs/intro/install/))

Verify that you have:
- Access to the Kubernetes cluster
- Persistent storage is enabled in the Kubernetes cluster, which has a default storage class setup. You can [change the default StorageClass](https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/).
- Access to a Google Cloud Platform storage bucket (see [instructions](https://cloud.google.com/storage/docs/creating-buckets))
- Optional: DNS service works in the Kubernetes cluster
- Optional: An ingress controller is set up in the Kubernetes cluster, for example [ingress-nginx](https://kubernetes.github.io/ingress-nginx/)

>**NOTE**: If you want to access Tempo from outside of the Kubernetes cluster, you will need an ingress. Ingress-related procedures are marked as optional.

<!-- This section should be verified before being made visible. It’s from Mimir and might need to be updated for Tempo.

## Security setup

This installation will not succeed if you have enabled the [PodSecurityPolicy](*https://v1-23.docs.kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#podsecuritypolicy) admission controller or if you are enforcing the Restricted policy with [Pod Security](https://v1-24.docs.kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-admission-labels-for-namespaces) admission controller. The reason is that the installation includes a deployment of MinIO. The [minio/minio chart](https://github.com/minio/minio/tree/master/helm/minio) is not compatible with running under a Restricted policy or the PodSecurityPolicy that the mimir-distributed chart provides.

If you are using the PodSecurityPolicy admission controller, then it is not possible to deploy the mimir-distributed chart with MinIO. Refer to Run Grafana Mimir in production using the Helm chart for instructions on setting up an external object storage and disable the built-in MinIO deployment with minio.enabled: false in the Helm values file.

If you are using the Pod Security admission controller, then MinIO and the tempo-distributed chart can successfully deploy under the baseline pod security level.
-->

## Install the Helm chart in a custom namespace

Using a custom namespace solves problems later on because you do not have to overwrite the default namespace.

1. Create a unique Kubernetes namespace, for example tempo-test:

```bash
kubectl create namespace tempo-test
```

1. For more details, see the Kubernetes documentation about [Creating a new namespace](https://kubernetes.io/docs/tasks/administer-cluster/namespaces/#creating-a-new-namespace).

1. Set up a Helm repository using the following commands:

   ```bash
   helm repo add grafana https://grafana.github.io/helm-charts
   helm repo update
   ```

   >**NOTE**: The Helm chart at https://grafana.github.io/helm-charts is a publication of the source code at grafana/tempo.

### Set Google Cloud Platform as the storage option

Before you run the Helm chart, you need to configure where trace data will be stored. These steps create a YAML file with the storage configuration.

You need the `secret_key` from your service provider used as part of the user or account authentication. In this case, it refers to S3’s secret key. Google Cloud Storage has a similar option.

1. Create a YAML file of Helm values called `custom.yaml`.

1. Add the following configuration to the file to set Google Cloud Platform as the trace storage.
   - Change `temorockstracing` to your `secret_key`.
   - Change `bucket` to match the bucket name in GCP.
   - Change `endpoint` to the correct one for your set up.

   ```yaml
   ---
   minio:
     enabled: false
   storage:
     trace:
       backend: s3
       s3:
         access_key: 'tempo-test'
         secret_key: 'temporockstracing'
         bucket: 'tempo-test'
         endpoint: 'stodev0.znet:9000'
         insecure: true
   ```

### Optional: Add custom configurations

You can use a YAML file, like `custom.yaml`, to store custom configuration options that override the defaults present in the Helm chart.

To see all of the configurable parameters for the `tempo-distributed` Helm chart, use the following command:

```bash
helm show values grafana/tempo-distributed
```

The configuration sections are added to the `custom.yaml` file. This file is included when you install or upgrade the Helm chart.

#### Optional: Configure an ingress

An ingress lets you externally access a Kubernetes cluster.
Replace `<ingress-host>` with a suitable hostname that DNS can resolve to the external IP address of the Kubernetes cluster.
For more information, see [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

>**NOTE**: On Linux systems, and if it is not possible for you set up local DNS resolution, you can use the `--add-host=<ingress-host>:<kubernetes-cluster-external-address>` command-line flag to define the `<ingress-host>` local address for the docker commands in the examples that follow.

1. Open your `custom.yaml` or create a YAML file of Helm values called `custom.yaml`.
1. Add the following configuration to the file:
   ```
   nginx:
     ingress:
       enabled: true
       ingressClassName: nginx
       hosts:
         - host: <ingress-host>
           paths:
             - path: /
               pathType: Prefix
       tls:
         # empty, disabled.
   ```

### Optional: Configure storage

You can use storage options, including cloud-based storage, like Google Cloud Platform (GCP), Amazon S3, and local storage options, like MinIO. The Helm chart in this guide uses GCP, however, you can select other storage options.
Refer to the []`storage` configuration block](https://grafana.com/docs/tempo/latest/configuration/#storage) for information on other storage options.

## Install Grafana Tempo using the Helm chart

Use the following command to install Tempo using the configuration options you’ve specified in the `custom.yaml` file:

```bash
helm -n tempo-test install tempo grafana/tempo-distributed -f custom.yaml
```

>**NOTE**: The output of the command contains the write and read URLs necessary for the following steps.

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

Check the statuses of the Tempo pods:

```
kubectl -n tempo-test get pods
```

The results look similar to this:

```
❯ kubectl -n tempo-test get pods
NAME                                	READY   STATUS	RESTARTS   AGE
tempo-compactor-6cd48b65cb-5hkws    	1/1 	Running   0      	79m
tempo-distributor-965fc4564-cq28j   	1/1 	Running   0      	79m
tempo-ingester-0                    	1/1 	Running   0      	77m
tempo-ingester-1                    	1/1 	Running   0      	78m
tempo-ingester-2                    	1/1 	Running   0      	79m
tempo-memcached-0                   	1/1 	Running   0      	94m
tempo-querier-54bfd999c7-z59bb      	1/1 	Running   0      	79m
tempo-query-frontend-757c66d7bd-rxtbl   1/1 	Running   0      	79m
```

Wait until all of the pods have a status of Running or Completed, which might take a few minutes.

## Next step
The next step is to test your Tempo installation by sending trace data to Grafana. You can use the [Set up a test application for a Tempo cluster](https://grafana.com/docs/tempo/latest/setup/set-up-test-app/) document for step-by-step instructions.

If you already have Grafana available, you can add a Tempo data source using the URL fitting to your environment. For example:
`http://tempo-query-frontend.trace-test.svc.cluster.znet:3100`