---
title: Too many jobs in the queue
weight: 473
---

# I am getting error message ‘Too many jobs in the queue’

You may see this error if the compactor isn’t running and the blocklist size has exploded. 
Possible reasons why the compactor may not be running are:

- Insufficient permissions.
- Compactor sitting idle because no block is hashing to it.
- Incorrect configuration settings.
### Diagnosing the issue
- Check metric `tempodb_compaction_bytes_written_total`
If this is greater than zero (0), it means the compactor is running and writing to the backend.
- Check metric `tempodb_compaction_errors_total`
If this metric is greater than zero (0), check the logs of the compactor for an error message.

#### Solutions
- Verify that the Compactor has the LIST, GET, PUT, and DELETE permissions on the bucket objects.
  - If these permissions are missing, assign them to the compactor container.
  - For detailed information, check - https://grafana.com/docs/tempo/latest/configuration/s3/#permissions
- If there’s a compactor sitting idle while others are running, port-forward to the compactor’s http endpoint. Then go to `/compactor/ring` and click **Forget** on the inactive compactor.
- Check the following configuration parameters to ensure that there are correct settings:
  - `max_block_bytes` to determine when the ingester cuts blocks. A good number is anywhere from 100MB to 2GB depending on the workload.
  - `max_compaction_objects` to determine the max number of objects in a compacted block. This should relatively high, generally in the millions.
  - `retention_duration` for how long traces should be retained in the backend.
