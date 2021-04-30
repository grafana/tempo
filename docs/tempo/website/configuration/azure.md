---
title: Azure Storage Account (Azure)
---

# Azure Storage Account (Azure) configuration
Azure backend is configured in the storage block. Tempo requires a dedicated bucket since it maintains a top-level object structure and does not support a custom prefix to nest within a shared bucket.

```
storage:
    trace:
        backend: azure                              # store traces in azure
        azure:
            container-name: tempo                   # store traces in this container.
            endpoint-suffix: blob.core.windows.net  # optional. Azure endpoint to use, defaults to Azure global(core.windows.net) for other
                                                    #           regions this needs to be changed e.g Azure China(blob.core.chinacloudapi.cn),
                                                    #           Azure German(blob.core.cloudapi.de), Azure US Government(blob.core.usgovcloudapi.net).
            storage-account-name: ""                # Name of the azure storage account
            storage-account-key: ""                 # optional. access key when using access key credentials.
```