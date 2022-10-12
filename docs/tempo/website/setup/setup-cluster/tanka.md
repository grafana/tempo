---
title: Deploy on Kubernetes with Tanka
weight: 200
---

# Deploy on Kubernetes with Tanka

Using this deployment guide, you can deploy Tempo to Kubernetes using a Jsonnet library and [Grafana Tanka](https://tanka.dev) to create a development cluster or sand-boxed environment.
This procedure uses MinIO to provide object storage regardless of the Cloud platform or on-premise storage you use.
In a production environment, you can use your cloud provider’s object storage service to avoid the operational overhead of running object storage in production.

This demo configuration includes [metrics-generator](https://grafana.com/docs/tempo/next/configuration/#metrics-generator).

To set up Tempo using Kubernetes with Tanka, you need to:

1. Configure Kubernetes and install Tanka
1. Set up the Tanka environment
1. Install libraries
1. Deploy MinIO object storage and configure metrics-generator
1. Deploy Tempo with the Tanka command

>**Note**: This configuration is not suitable for a production environment but can provide a useful way to learn about Tempo.

## Before you begin

To deploy Tempo to Kubernetes with Tanka, you need:

  * A Kubernetes cluster with at least 40 CPUs and 46GB of memory for the default configuration. Small ingest or query volumes could use a far smaller configuration.
  * `kubectl`


## Configure Kubernetes and install Tanka

The first step is to configure Kubernetes and install Tanka.

1. Create a new directory for the installation, and make it your current working directory:

    ```bash
    mkdir tempo
    cd tempo
    ```

1. Create a Kubernetes namespace. You can use any namespace that you wish; this example uses `tempo`.

    ```bash
    kubectl create namespace tempo
    ```

1. Install Grafana Tanka; refer to [Installing Tanka](https://tanka.dev/install).

1. Install `jsonnet-bundler`; refer to the [`jsonnet-bundler` README](https://github.com/jsonnet-bundler/jsonnet-bundler/#install).

## Set up the Tanka environment

Tanka requires the current context for your Kubernetes environment.

1. Check the current context for your Kubernetes cluster and ensure it's correct:
   ```bash
   kubectl config current-context
   ```

2. Initialize Tanka. This will use the current Kubernetes context:

   ```bash
   tk init --k8s=false
   tk env add environments/tempo
   tk env set environments/tempo \
    --namespace=tempo \
    --server-from-context=$(kubectl config current-context)
   ```

## Install libraries

Install the `k.libsonnet`, Jsonnet, and Memcachd libraries.

1. Install `k.libsonnet` for your version of Kubernetes:

   ```bash
   mkdir -p lib
   export K8S_VERSION=1.22
   jb install github.com/jsonnet-libs/k8s-libsonnet/${K8S_VERSION}@main
   cat <<EOF > lib/k.libsonnet
   import 'github.com/jsonnet-libs/k8s-libsonnet/${K8S_VERSION}/main.libsonnet'
   EOF
   ```

1. Install the Tempo Jsonnet library and its dependencies.

    ```bash
    jb install github.com/grafana/tempo/operations/jsonnet/microservices@main
    ```

1. Install the Memcached library and its dependencies.

    ```bash
    jb install github.com/grafana/jsonnet-libs/memcached@master
    ```

## Deploy MinIO object storage

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
                - "mkdir -p /storage/tempo-data"
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
        kubectl apply --namespace tempo -f minio.yaml
        ```

  1. To check that MinIO is correctly configured, sign in to MinIO and verify that a bucket has been created. Without these buckets, no data will be stored.
     1.  Port-forward MinIO to port 9001:

         ```bash
          kubectl port-forward --namespace tempo service/minio 9001:9001
          ```
     1. Navigate to the MinIO admin bash using your browser: `https://localhost:9001`. The sign-in credentials are username `minio` and password `minio123`.
     1. Verify that the Buckets page lists `tempo-data`.

1. Configure the Tempo cluster using the MinIO object storage by updating the contents of the `environments/tempo/main.jsonnet` file by running the following command:

   ```bash
   cat <<EOF > environments/tempo/main.jsonnet
   // The jsonnet file used to generate the Kubernetes manifests.
   local tempo = import 'microservices/tempo.libsonnet';

   tempo {
       _images+:: {
           tempo: 'grafana/tempo:latest',
           tempo_query: 'grafana/tempo-query:latest',
       },

       _config+:: {
           namespace: 'tempo',
           metrics_generator_enabled: true,

           compactor+: {
               replicas: 1,
           },
           query_frontend+: {
               replicas: 2,
           },
           querier+: {
               replicas: 3,
           },
           ingester+: {
               replicas: 3,
               pvc_size: '10Gi',
               pvc_storage_class: 'fast',
           },
           distributor+: {
               replicas: 3,
               receivers: {
                   jaeger: {
                       protocols: {
                           grpc: {
                               endpoint: '0.0.0.0:14250',
                           },
                       },
                   },
                   otlp: {
                       protocols: {
                           grpc: {
                               endpoint: '0.0.0.0:4317',
                           },
                       },
                   },
               },
           },

           metrics_generator+: {
               replicas: 1,
               ephemeral_storage_request_size: '10Gi',
               ephemeral_storage_limit_size: '11Gi',
           },
           memcached+: {
               replicas: 3,
           },

           bucket: 'tempo-data',
           backend: 's3',
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
           metrics_generator+: {
               processor: {
                   span_metrics: null,
                   service_graphs: null,
               },

               registry+: {
                   external_labels: {
                       source: 'tempo',
                   },
               },
           },
           overrides+: {
               max_search_bytes_per_trace: 5000000,
               metrics_generator_processors: ['service-graphs', 'span-metrics'],
           },
       },

       local statefulSet = $.apps.v1.statefulSet,
       tempo_ingester_statefulset+:
           statefulSet.mixin.spec.withPodManagementPolicy('Parallel'),
   }
   EOF
   ```

### Enable metrics-generator

Note that we've enabled metrics generation in the above configuration, but we haven't actually specified anywhere to send the generated metrics data. If you'd like to remote write these metrics onto a Prometheus compatible instance (such as Grafana Cloud Metrics or a Mimir instance), you'll need to include the following configuration block in the `metrics_generator` section of the `tempo_config` block above (this assumes basic auth is required, if not then remove the `basic_auth` section):

```json
storage+: {
    remote_write: [
        {
            url: 'https://<urlForPrometheusCompatibleStore>/api/v1/write',
            send_exemplars: true,
            basic_auth: {
                username: '<username>',
                password: '<password>',
            },
        }
    ],
},
```

## Deploy Tempo using Tanka

1. Deploy Tempo using the Tanka command:
    ```bash
    tk apply environments/tempo/main.jsonnet
    ```

> **Note**: If the ingesters don’t start after deploying Tempo with the Tanka command, this may be related to the storage class selected for the Write Ahead Logs. If this is the case, add an appropriate storage class to the ingester configuration. For example, to add a standard instead of fast storage class, add the following to the `config` (not `tempo_config`) section in the previous step:
>
  ```bash
    ingester+: {
      pvc_storage_class: 'standard',
    },
  ```

## Next steps

The Tempo instance will now accept the two configured trace protocols (OTLP gRPC and Jaeger gRPC) via the distributor service at `distributor.tempo.svc.cluster.local` on the relevant ports:

* OTLP gRPC: `4317`
* Jaeger gRPC: `14250`

You can query Tempo using the `query-frontend.tempo.svc.cluster.local` service on port `3200` for Tempo queries or port `16686` or `16687` for Jaeger type queries.

Now you have a Tempo cluster, it's worth getting some test data into it.