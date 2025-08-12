---
title: Azure blob storage permissions and management
menuTitle: Azure blob storage
description: Azure blob storage permissions and configuration options for Tempo.
aliases:
  - ../../configuration/azure/ # /docs/tempo/<TEMPO_VERSION>/configuration/azure/
---

# Azure blob storage permissions and management

Tempo supports Azure blob storage for both monolithic and distributed modes.
Some of the supported features include:

- Object layout: custom `container_name` and optional `prefix` to nest objects in a shared container.
- Performance: hedged requests (`hedge_requests_at`, `hedge_requests_up_to`) to reduce long-tail latency.
- Regional or sovereign clouds: configurable endpoint suffix (for example, US Gov, Germany) using `endpoint_suffix`.
- Local development: Azurite emulator support (non-`blob.\*` endpoint style is auto-detected).
- Ops guidance: compatible with Azure Storage lifecycle policies for cleanup (example provided in the doc).

Tempo supports the following authentication methods:

- Shared key
- Managed Identity (system/user-assigned) [use `use_managed_identity`, `user_assigned_id`]
- Azure Workload Identity (federated token) [use `use_federated_token`]

## Before you begin

Tempo requires the following configuration to authenticate to and access Azure blob storage:

- Storage Account name specified in the configuration file as `storage_account_name` or in the environment variable `AZURE_STORAGE_ACCOUNT`.
- Credentials for accessing the Storage Account that are one of the following:

  - Storage Account access key specified in the configuration file as `storage_account_key` or in the environment variable `AZURE_STORAGE_KEY`.
  - An Azure Managed Identity that is either system or user assigned. To use Azure Managed Identities, you need to set `use_managed_identity` to `true` in the configuration file or set `user_assigned_id` to the client ID for the managed identity you'd like to use.
    - For a system-assigned managed identity, no additional configuration is required.
    - For a user-assigned managed identity, you need to set `user_assigned_id` to the client ID for the managed identity in the configuration file.
  - Via Azure Workload Identity. To use Azure Workload Identity, you need to enable Azure Workload Identity on your cluster, add the required label and annotation to the service account and the required Pod label. Additionally, you need to set `use_federated_token` to `true` to utilize Azure Workload Identity.

- If you are using Azure sovereign or regional clouds (for example, Azure US Government or Azure Germany), set the `endpoint_suffix` under the Azure storage configuration to the appropriate domain. See examples below.

## Sample configuration for Tempo monolithic mode

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

Here is an example configuration using Azure Workload Identity.

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

## Sample configuration for Tempo distributed mode

In Distributed mode the `trace` configuration needs to be applied against the `storage` object, which resides at the root of the Values object. Additionally, the `extraArgs` and `extraEnv` configuration need to be applied to each of the following services:

- `distributor`
- `compactor`
- `ingester`
- `querier`
- `queryFrontend`

```yaml
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

## Additional configuration options

The following sections provide additional configuration options for Azure blob storage.

### Use Azurite for local development

You can use the Azurite emulator to test your Tempo configuration locally.
Refer to the [Azurite emulator documentation](https://learn.microsoft.com/en-us/azure/storage/common/storage-use-azurite) for more details.

Tempo treats any Azure `endpoint_suffix` that doesn't start with `blob.` as Azurite and automatically switches to the emulator URL style.
For more information about the Azurite URL style, refer to the [Azure Storage documentation](https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob#emulated-storage-service-uri).

Set `backend` to `azure`, supply your Azurite account and key, and point `endpoint_suffix` to the emulator `host:port`.
Tempo handles the Azurite URL format automatically.

If you encounter any issues, try using the fully qualified domain name (FQDN) for the Azurite emulator.
For example, `azurite-host.azure.local:10000`.

In this example, replace the example values with your Azure configuration values.

```yaml
storage:
  trace:
    blocklist_poll: 1s
    backend: azure
    azure:
      container_name: container-name # how to store data in azure
      endpoint_suffix: azurite-host.svc.cluster.local:10000 # Azurite emulator host:port
      storage_account_name: "storage-account-name"
      storage_account_key: "STORAGE_ACCOUNT_ACCESS_KEY"
```

### Endpoints for regional and sovereign clouds

By default, Tempo connects to `blob.core.windows.net`. For Azure sovereign clouds, you must specify `endpoint_suffix` to match the correct environment. For example:

- Azure US Government: `blob.core.usgovcloudapi.net`
- Azure Germany: `blob.core.cloudapi.de`

Set this value under `storage.trace.azure.endpoint_suffix` in your configuration.

#### Troubleshoot Azure sovereign clouds

If you are using Azure US Government or Azure Germany and see connectivity or authentication errors against `blob.core.windows.net`, set `storage.trace.azure.endpoint_suffix` to the correct domain for your environment.
For example, `blob.core.usgovcloudapi.net` for Azure US Government or `blob.core.cloudapi.de` for Azure Germany.

### Azure blocklist polling

If you are hosting Tempo on Azure, you may need to update two values to ensure consistent successful blocklist polling.
If you experience [this issue](https://stackoverflow.com/questions/12917857/the-specified-block-list-is-invalid-while-uploading-blobs-in-parallel/55902744#55902744), try setting `blocklist_poll_tenant_index_builders` to 1.

Additionally, if you are seeing DNS failures like the ones below, try increasing `blocklist_poll_jitter_ms`.
Refer to the discussion in [GitHub issue 1462](https://github.com/grafana/tempo/issues/1462).

For example:

```text
reading storage container: Head "https://tempoe**************.blob.core.windows.net/tempo/single-tenant/d8aafc48-5796-4221-ac0b-58e001d18515/meta.compacted.json?timeout=61": dial tcp: lookup tempoe**************.blob.core.windows.net on 10.0.0.10:53: dial udp 10.0.0.10:53: operation was canceled
```

Your final configuration may look something like:

```yaml
storage:
  trace:
    blocklist_poll_tenant_index_builders: 1
    blocklist_poll_jitter_ms: 500
```

### (Optional) Storage Account management policy for cleaning up the storage container

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
            "blobTypes": ["blockBlob"],
            "prefixMatch": ["tempo-data"]
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
            "blobTypes": ["blockBlob"],
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
