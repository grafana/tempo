---
title: Object storage
description: Setup for storing traces to Object Storage
menuTitle: Object storage
weight: 200
aliases:
- /docs/tempo/operator/object-storage
---

# Object storage

Tempo Operator supports [AWS S3](https://aws.amazon.com/), [Azure](https://azure.microsoft.com), [GCS](https://cloud.google.com/), [Minio](https://min.io/) and [OpenShift Data Foundation](https://www.redhat.com/en/technologies/cloud-computing/openshift-data-foundation) for TempoStack object storage.

## AWS S3

### Requirements

* Create a [bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html) on AWS.

### Static token installation

1. Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-s3 \
      --from-literal=bucket="<BUCKET_NAME>" \
      --from-literal=endpoint="<AWS_BUCKET_ENDPOINT>" \
      --from-literal=access_key_id="<AWS_ACCESS_KEY_ID>" \
      --from-literal=access_key_secret="<AWS_ACCESS_KEY_SECRET>"
    ```

  where `tempostack-dev-s3` is the secret name.

2. Create an instance of TempoStack by referencing the secret name and type as `s3`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-s3
        type: s3
  ```

### AWS Security Token Service (STS) installation

1. Create a custom AWS IAM Role associated with a trust relationship to Tempo's Kubernetes `ServiceAccount`:
  
  ```yaml
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": {
          "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER}"
        },
        "Action": "sts:AssumeRoleWithWebIdentity",
        "Condition": {
          "StringEquals": {
            "${OIDC_PROVIDER}:sub": [
              "system:serviceaccount:${TEMPOSTACK_NS}:tempo-${TEMPOSTACK_NAME}",
              "system:serviceaccount:${TEMPOSTACK_NS}:tempo-${TEMPOSTACK_NAME}-query-frontend"
           ]
         }
       }
     }
    ]
  }
  ```
  
2. Create an AWS IAM role:

  ```yaml
  aws iam create-role \
    --role-name "tempo-s3-access" \
    --assume-role-policy-document "file:///tmp/trust.json" \
    --query Role.Arn \
    --output text
  ```

3. Attach a specific policy to that role:

  ```yaml 
  aws iam attach-role-policy \
    --role-name "tempo-s3-access" \
    --policy-arn "arn:aws:iam::aws:policy/AmazonS3FullAccess"
  ```

4. Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-s3 \
      --from-literal=bucket="<BUCKET_NAME>" \
      --from-literal=region="<AWS_REGION>" \
      --from-literal=role_arn="<ROLE ARN>"
    ```

where `tempostack-dev-s3` is the secret name.

5. Create an instance of TempoStack by referencing the secret name and type as `s3`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-s3
        type: s3
  ```

## Azure

### Requirements

* Create a [bucket](https://docs.microsoft.com/en-us/azure/storage/blobs/storage-blobs-introduction) on Azure.

### Installation

1. Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-azure \
      --from-literal=container="<AZURE_CONTAINER_NAME>" \
      --from-literal=account_name="<AZURE_ACCOUNT_NAME>" \
      --from-literal=account_key="<AZURE_ACCOUNT_KEY>"
    ```

  where `tempostack-dev-azure` is the secret name.

2. Create an instance of TempoStack by referencing the secret name and type as `azure`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-azure
        type: azure
  ```

## Google Cloud Storage

### Requirements

* Create a [project](https://cloud.google.com/resource-manager/docs/creating-managing-projects) on Google Cloud Platform.
* Create a [bucket](https://cloud.google.com/storage/docs/creating-buckets) under same project.
* Create a [service account](https://cloud.google.com/docs/authentication/getting-started#creating_a_service_account) under same project for GCP authentication.

### Installation

1. Copy the service account credentials received from GCP into a file name `key.json`.
2. Create an Object Storage secret with keys `bucketname` and `key.json` as follows:

    ```console
    kubectl create secret generic tempostack-dev-gcs \
      --from-literal=bucketname="<BUCKET_NAME>" \
      --from-file=key.json="<PATH/TO/KEY.JSON>"
    ```

  where `tempostack-dev-gcs` is the secret name, `<BUCKET_NAME>` is the name of bucket created in requirements step and `<PATH/TO/KEY.JSON>` is the file path where the `key.json` was copied to.

3. Create an instance of TempoStack by referencing the secret name and type as `gcs`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-gcs
        type: gcs
  ```

## MinIO

### Requirements

* Deploy MinIO on your cluster, e.g. using the [MinIO Operator](https://operator.min.io/) or another method.

* Create a [bucket](https://docs.min.io/docs/minio-client-complete-guide.html) on MinIO using the CLI.

### Installation

1. Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-minio \
      --from-literal=bucket="<BUCKET_NAME>" \
      --from-literal=endpoint="<MINIO_BUCKET_ENDPOINT>" \
      --from-literal=access_key_id="<MINIO_ACCESS_KEY_ID>" \
      --from-literal=access_key_secret="<MINIO_ACCESS_KEY_SECRET>"
    ```

  where `tempostack-dev-minio` is the secret name.

2. Create an instance of TempoStack by referencing the secret name and type as `s3`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-minio
        type: s3
  ```

## OpenShift Data Foundation

### Requirements

* Deploy the [OpenShift Data Foundation](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation/4.10) on your cluster.

* Create a bucket via an ObjectBucketClaim.

### Installation

1. Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-odf \
      --from-literal=bucket="<BUCKET_NAME>" \
      --from-literal=endpoint="https://s3.openshift-storage.svc" \
      --from-literal=access_key_id="<ACCESS_KEY_ID>" \
      --from-literal=access_key_secret="<ACCESS_KEY_SECRET>"
    ```

  where `tempostack-dev-odf` is the secret name. You can copy the values for `BUCKET_NAME`, `ACCESS_KEY_ID` and `ACCESS_KEY_SECRET` from your ObjectBucketClaim's accompanied secret.

2. Create an instance of TempoStack by referencing the secret name and type as `s3`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-odf
        type: s3
  ```
