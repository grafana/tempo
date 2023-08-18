---
title: Response larger than the max
description: Troubleshoot response larger than the max error message
weight: 477
aliases:
- ../operations/troubleshooting/response-too-large/
---

# Response larger than the max

The error message will take a similar form to the following:

```
500 Internal Server Error Body: response larger than the max (<size> vs <limit>)
```

This error indicates that the response received or sent is too large.
This can happen in multiple places, but it's most commonly seen in the query path,
with messages between the querier and the query frontend.

## Solutions

### Tempo server (general)

Tempo components communicate with each other via gRPC requests.
To increase the maximum message size, you can increase the gRPC message size limit in the server block.

```yaml
server:
  grpc_server_max_recv_msg_size: <size>
  grpc_server_max_send_msg_size: <size>
```

The server config block is not synchronized across components.
Most likely you will need to increase the message size limit in multiple components.

### Querier

Additionally, querier workers can be configured to use a larger message size limit.

```yaml
querier:
    frontend_worker:
        grpc_client_config:
            max_send_msg_size: <size>
```

### Ingestion

Lastly, message size is also limited in ingestion and can be modified in the distributor block.

```yaml
distributor:
  receivers:
    otlp:
      grpc:
        max_recv_msg_size_mib: <size>
```
