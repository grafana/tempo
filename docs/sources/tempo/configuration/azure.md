---
title: Azure blob storage permissions and management
menuTitle: Azure blob storage
description: Azure blog storage permissions and configuration options for Tempo.
weight: 30
---

# Azure blob storage permissions and management

Tempo requires the following configuration to authenticate to and access Azure blob storage:

- Storage Account name specified in the configuration file as `storage_account_name` or in the environment variable `AZURE_STORAGE_ACCOUNT`
- Credentials for accessing the Storage Account that are one of the following:
  - Storage Account access key specified in the configuration file as `storage_account_key` or in the environment variable `AZURE_STORAGE_KEY`
  - An Azure Managed Identity that is either system or user assigned. To use Azure Managed Identities, you'll need to set `use_managed_identity` to `true` in the configuration file or set `user_assigned_id` to the client ID for the managed identity you'd like to use.
      - For a system-assigned managed identity, no additional configuration is required.
      - For a user-assigned managed identity, you'll need to set `user_assigned_id` to the client ID for the managed identity in the configuration file.
  - Via Azure Workload Identity. To use Azure Workload Identity, you'll need to enable Azure Workload Identity on your cluster, add the required label and annotation to the service account and the required pod label. Additionally, you will need to set `use_federated_token` to `true` to utilize Azure Workload Identity.

## Sample configuration (for Tempo Monolithic Mode)

### Access key
This sample configuration shows how to set up Azure blob storage using Helm charts and an access key from Kubernetes secrets.

```yaml
tempo:
  storage:
    trace:
      backend: azure
      azure:
        container_name: container-name
        storage_account_name: storage-account-name
        storage_account_key: ${STORAGE_ACCOUNT_ACCESS_KEY}

  extraArgs:
    config.expand-env: true
  extraEnv:
    - name: STORAGE_ACCOUNT_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: secret-name
          key: STORAGE_ACCOUNT_ACCESS_KEY
```

### Azure Workload Identity
Here is an example config for using Azure Workload Identity. 
```yaml
tempo:
  storage:
    trace:
      backend: azure
      azure:
        container_name: container-name
        storage_account_name: storage-account-name
        use_federated_token: true
```

## Sample configuration (for Tempo Distributed Mode)

In Distributed mode the `trace` configuration needs to be applied against the `storage` object, which resides at the root of the Values object. Additionally, the `extraArgs` and `extraEnv` configuration need to be applied to each of the following services:
 - distributor
 - compactor
 - ingester
 - querier
 - queryFrontend

```
storage:
  trace:
    backend: azure
    azure:
      container_name: tempo-traces
      storage_account_name: stgappgeneraluks
      storage_account_key: ${STORAGE_ACCOUNT_ACCESS_KEY}


distributor:
  extraArgs:
  - "-config.expand-env=true"
  extraEnv:
  - name: STORAGE_ACCOUNT_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: tempo-traces-stg-key
        key: tempo-traces-key

compactor:
  extraArgs:
  - "-config.expand-env=true"
  extraEnv:
  - name: STORAGE_ACCOUNT_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: tempo-traces-stg-key
        key: tempo-traces-key

ingester:
  extraArgs:
  - "-config.expand-env=true"
  extraEnv:
  - name: STORAGE_ACCOUNT_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: tempo-traces-stg-key
        key: tempo-traces-key

querier:
  extraArgs:
  - "-config.expand-env=true"
  extraEnv:
  - name: STORAGE_ACCOUNT_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: tempo-traces-stg-key
        key: tempo-traces-key

queryFrontend:
  extraArgs:
  - "-config.expand-env=true"
  extraEnv:
  - name: STORAGE_ACCOUNT_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: tempo-traces-stg-key
        key: tempo-traces-key
```

## Azure blocklist polling

If you are hosting Tempo on Azure, two values may need to be updated to ensure consistent successful blocklist polling. If you are
experiencing [this issue](https://stackoverflow.com/questions/12917857/the-specified-block-list-is-invalid-while-uploading-blobs-in-parallel/55902744#55902744), try setting `blocklist_poll_tenant_index_builders` to 1.

Additionally, if you are seeing DNS failures like the ones below, try increasing `blocklist_poll_jitter_ms`. Discussion [here](https://github.com/grafana/tempo/issues/1462).
```
reading storage container: Head "https://tempoe**************.blob.core.windows.net/tempo/single-tenant/d8aafc48-5796-4221-ac0b-58e001d18515/meta.compacted.json?timeout=61": dial tcp: lookup tempoe**************.blob.core.windows.net on 10.0.0.10:53: dial udp 10.0.0.10:53: operation was canceled
```

Your final configuration may look something like:
```
  storage:
    trace:
      blocklist_poll_tenant_index_builders: 1
      blocklist_poll_jitter_ms: 500
```

## (Optional) Storage Account management policy for cleaning up the storage container

The following Storage Account management policy shows an example of cleaning up
files from the container after they have been deleted for a period of time.

```json
{
  "id": "/subscriptions/00000000-0000-0000000000000000000000/resourceGroups/resourceGroupName/providers/Microsoft.Storage/storageAccounts/accountName/managementPolicies/default",
  "lastModifiedTime": "2021-11-30T19:19:54.855455+00:00",
  "name": "DefaultManagementPolicy",
  "policy": {
    "rules": [
      {
        "definition": {
          "actions": {
            "baseBlob": {
              "delete": {
                "daysAfterLastAccessTimeGreaterThan": null,
                "daysAfterModificationGreaterThan": 60.0
              },
              "enableAutoTierToHotFromCool": null,
              "tierToArchive": null,
              "tierToCool": null
            },
            "snapshot": null,
            "version": null
          },
          "filters": {
            "blobIndexMatch": null,
            "blobTypes": [
              "blockBlob"
            ],
            "prefixMatch": [
              "tempo-data"
            ]
          }
        },
        "enabled": true,
        "name": "TempoBlobRetention",
        "type": "Lifecycle"
      },
      {
        "definition": {
          "actions": {
            "baseBlob": null,
            "snapshot": null,
            "version": {
              "delete": {
                "daysAfterCreationGreaterThan": 7.0
              },
              "tierToArchive": null,
              "tierToCool": null
            }
          },
          "filters": {
            "blobIndexMatch": null,
            "blobTypes": [
              "blockBlob"
            ],
            "prefixMatch": []
          }
        },
        "enabled": true,
        "name": "VersionRetention",
        "type": "Lifecycle"
      }
    ]
  },
  "resourceGroup": "resource-group-name",
  "type": "Microsoft.Storage/storageAccounts/managementPolicies"
}
```
