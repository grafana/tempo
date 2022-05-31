---
title: Azure Permissions and Management
weight: 3
---

# Azure Blob Storage Permissions and Management

In order for Tempo to authenticate to and access Azure blob storage, the following configuration is required:

- Storage Account name specified in the configuration file as `storage-account-name` or in the environment variable `AZURE_STORAGE_ACCOUNT`
- Credentials for accessing the Storage Account; can be one of the following
  - Storage Account access key specified in the configuration file as `storage-account-key` or in the environment variable `AZURE_STORAGE_KEY`
  - An Azure Managed Identity; either system or user assigned. To use Azure Managed Identities, you'll need to set `use-managed-identity` to `true` in the configuration file 
      - For a system-assigned managed identity, no additional configuration is required.
      - For a user-assigned managed identity, you'll need to set `user-assigned-id` to the client ID for the managed identity in the configuration file.

## (Optional) Storage Account Management Policy for Cleaning Up the Storage Container

The following storage account management policy shows an example of cleaning up
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
