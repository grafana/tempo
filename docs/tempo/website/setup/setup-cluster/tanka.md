---
title: Deploy on Kubernetes with Tanka
weight: 200
---

# Deploy on Kubernetes with Tanka

Using this deployment guide, you can deploy Tempo to Kubernetes using a Jsonnet library and [Grafana Tanka](https://tanka.dev) to create a development cluster or sandboxed environment. 
This procedure uses MinIO to provide object storage regardless of the Cloud platform or on-premise storage you use. 
In a production environment, you can use your cloud provider’s object storage service to avoid the operational overhead of running object storage in production.

This demo configuration does not include [metrics-generator](https://grafana.com/docs/tempo/next/configuration/#metrics-generator). 

>**Note**: This configuration is not suitable for a production environment but can provide a useful way to learn about GET.

## Before you begin

To deploy Tempo to Kubernetes with Tanka, you need: 

  * A Kubernetes cluster with at least 40 CPUs and 46GB of memory for the default configuration. Small ingest or query volumes could use a far smaller configuration. 
  * `kubectl`

## Procedure

To set up Tempo using Kubernetes with Tanka, you need to:

1. Configure Kubernetes and install Tanka
1. Set up the Tanka environment
1. Install libraries
1. Deploy MinIO object storage
1. Deploy Tempo with the Tanka command

### Configure Kubernetes and install Tanka

The first step is to configure Kubernetes and install Tanka. 

1. Create a new directory for the installation, and make it your current working directory:

    ```bash
    mkdir get
    cd get
    ```

1. Create a Kubernetes namespace. You can use any namespace that you wish; this example uses `enterprise-traces`.

    ```bash
    kubectl create namespace enterprise-traces
    ```

1. Create a Kubernetes Secret for your GET license:

    ```bash
    kubectl --namespace=enterprise-traces create secret generic get-license --from-file=/path/to/license.jwt
    ```

1. Install Grafana Tanka; refer to [Installing Tanka](https://tanka.dev/install).

1. Install `jsonnet-bundler`; refer to the [`jsonnet-bundler` README](https://github.com/jsonnet-bundler/jsonnet-bundler/#install).

### Set up the Tanka environment

Tanka requires the current context for your Kubernetes environment.

1. Acquire the current context for your Kubernetes cluster:
   ```bash
   kubectl config current-context

1. Initialize Tanka. Replace `<KUBECFG CONTEXT NAME>` with the acquired context name. 

   ```bash
   tk init --k8s=false
   tk env add environments/enterprise-traces
   tk env set environments/enterprise-traces \
    --namespace=enterprise-traces \
    --server-from-context=<KUBECFG CONTEXT NAME>
   ```

### Install libraries

Install the `k.libsonnet`, Jsonnet, and Memcachd libraries.

1. Install `k.libsonnet` for your version of Kubernetes:

   ```bash
   mkdir -p lib
   export K8S_VERSION=1.21
   jb install github.com/jsonnet-libs/k8s-libsonnet/${K8S_VERSION}@main
   cat <<EOF > lib/k.libsonnet
   import 'github.com/jsonnet-libs/k8s-libsonnet/${K8S_VERSION}/main.libsonnet'
   EOF
   ```

1. Install the Tempo Jsonnet library and its dependencies.

    ```bash
    jb install github.com/grafana/tempo/operations/jsonnet/enterprise@main
    ```

1. Install the Memcached library and its dependencies.

    ```bash
    jb install github.com/grafana/jsonnet-libs/memcached@master
    ```

### Deploy MinIO object storage

[MinIO](https://min.io) is an open source Amazon S3-compatible object storage service that is freely available and easy to run on Kubernetes.

1. Create a file named `minio.yaml` and copy the following YAML configuration into it. You may need to remove/modify the `storageClassName` depending on your Kubernetes platform. GKE, for example, may not support `local-path` name but may support another option such as `standard`.

    ```yaml
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      # This name uniquely identifies the PVC. Will be used in deployment below.
      name: minio-pv-claim
      labels:
        app: minio-storage-claim
    spec:
      # Read more about access modes here: http://kubernetes.io/docs/user-guide/persistent-volumes/#access-modes
      accessModes:
        - ReadWriteOnce
      storageClassName: local-path
      resources:
        # This is the request for storage. Should be available in the cluster.
        requests:
          storage: 50Gi
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: minio
    spec:
      selector:
        matchLabels:
          app: minio
      strategy:
        type: Recreate
      template:
        metadata:
          labels:
            # Label is used as selector in the service.
            app: minio
        spec:
          # Refer to the PVC created earlier
          volumes:
            - name: storage
              persistentVolumeClaim:
                # Name of the PVC created earlier
                claimName: minio-pv-claim
          initContainers:
            - name: create-buckets
              image: busybox:1.28
              command:
                - "sh"
                - "-c"
                - "mkdir -p /storage/enterprise-traces-data && mkdir -p /storage/enterprise-traces-admin"
              volumeMounts:
                - name: storage # must match the volume name, above
                  mountPath: "/storage"
          containers:
            - name: minio
              # Pulls the default Minio image from Docker Hub
              image: minio/minio:latest
              args:
                - server
                - /storage
                - --console-address
                - ":9001"
              env:
                # Minio access key and secret key
                - name: MINIO_ACCESS_KEY
                  value: "minio"
                - name: MINIO_SECRET_KEY
                  value: "minio123"
              ports:
                - containerPort: 9000
                - containerPort: 9001
              volumeMounts:
                - name: storage # must match the volume name, above
                  mountPath: "/storage"
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: minio
    spec:
      type: ClusterIP
      ports:
        - port: 9000
          targetPort: 9000
          protocol: TCP
          name: api
        - port: 9001
          targetPort: 9001
          protocol: TCP
          name: console
      selector:
        app: minio
    ```

  1. Run the following command to apply the minio.yaml file: 

        ```bash
        kubectl apply --namespace enterprise-traces -f minio.yaml
        ```

  1. To check that MinIO is correctly configured, sign in to MinIO and verify that two buckets have been created. Without these buckets, no data will be stored. 
     1.  Port-forward MinIO to port 9001:

         ```bash
          kubectl port-forward --namespace enterprise-traces service/minio 9001:9001
          ```
     1. Navigate to the MinIO admin bash using your browser: `https://localhost:9001`. The sign-in credentials are username `minio` and password `minio123`. 
     1. Verify that the Buckets page lists `enterprise-traces-admin` and `enterprise-traces-data`. 

1. Configure the GET cluster using the MinIO object storage by replacing the contents of the `environments/enterprise-traces/main.jsonnet` file with the following configuration:

   ```bash
   cat <<EOF > environments/enterprise-traces/main.jsonnet
   local get = import 'github.com/grafana/tempo/operations/jsonnet/enterprise/main.libsonnet';

   get {
     _config+:: {
       namespace: 'enterprise-traces',
       bucket: 'enterprise-traces-data',
       backend: 's3',

       // Set to true the first time installing GET, this will create the tokengen job. Once this job
       // has run this settings should be deleted.
       create_tokengen_job: true,
       metrics_generator+: {
         ephemeral_storage_request_size: '0',
         ephemeral_storage_limit_size: '0',
       },
     },

     tempo_config+:: {
       storage+: {
         trace+: {
           s3: {
               bucket: $._config.bucket,
               access_key: 'minio',
               secret_key: 'minio123',
               endpoint: 'minio:9000',
               insecure: true,
           },
         },
       },
       admin_api+: {
         leader_election: {
           enabled: false,
         },
       },
       admin_client+: {
         storage+: {
           type: 's3',
             s3: {
               bucket_name: 'enterprise-traces-admin',
               access_key_id: 'minio',
               secret_access_key: 'minio123',
               endpoint: 'minio:9000',
               insecure: true,
             },
           },
       },
     },

     tempo_ingester_container+:: {
       securityContext+: {
         runAsUser: 0,
       },
     },

     // Deploy tokengen Job available on a first run.
     tokengen_job+::: {},
   }

   EOF
   ```

### Deploy Tempo using Tanka

1. Deploy Tempo using the Tanka command:
    ```bash
    tk apply environments/enterprise-traces/main.jsonnet
    ```

> **Note**: If the ingesters don’t start after deploying GET with the Tanka command, this may be related to the storage class selected for the Write Ahead Logs. If this is the case, add an appropriate storage class to the ingester configuration. For example, to add a standard instead of fast storage class, add the following to the `config` (not `tempo_config`) section in the previous step:
> 
  ```bash
    ingester+: {
      pvc_storage_class: 'standard',
    },
  ```
1. Retrieve the Tempo token. This can be achieved at examining the logs for the `tokengen` job:
   ```bash
    kubectl --namespace=enterprise-traces logs job.batch/tokengen --container tokengen
    ```
    You should see a line like:
    ```bash
      Token:  X19hZG1pbl9fLWU2Y2U5MTRkNzYzODljNDA6Mlg2My9gMzlcNy8sMjUrXF9YMDM9TWBD
    ```

1. Save this token. You will need it when setting up your tenants and Grafana Enterprise Traces plugin.

## Next steps

Refer to [Set up the GET plugin for Grafana]({{< relref "../setup-get-plugin-grafana" >}}) to integrate your GET cluster with Grafana and a UI to interact with the Admin API.