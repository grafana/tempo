---
aliases:
- /docs/tempo/v1.2.1/operations/ingester_pvcs/
title: Ingester PVCs
weight: 5
---

### Ingester Persistent Volume Operations

Tempo ingesters make heavy use of local disks to store write-ahead logs and blocks before being flushed to the backend (GCS/S3/etc).   It is important to monitor the free volume space as full disks can lead to data loss and other errors.  The amount of disk space available affects how much volume a Tempo ingester can process, and the length of time an outage to the backend can be tolerated.

Therefore it may be necessary to increase the disk space for ingesters as usage increases. When deployed as a StatefulSet with Persistent Volume Claims (PVC), some manual steps are required. The following has worked successfully on GKE with GCS:

1. Edit the persistent volume claim (pvc) for each ingester to the new size.

   ```
   kubectl patch pvc -n <namespace> -p '{"spec": {"resources": {"requests": {"storage": "'15Gi'"}}}}' <pod-name>
   ```

   Check all disks have been upgraded by running:

   `kubectl get pvc -n <namespace>`

   A restart is not necessary as the pods will automatically detect the increased disk space.

2. Delete the StatefulSet but leave the pods running:

   `kubectl delete sts --cascade=false -n <namespace> ingester`

3. Edit and recreate the Statefulset with the new size. This covers new pods.  There are many ways to deploy Tempo to kubernetes, these are examples for the popular ones:
    * Raw yaml: `kubectl apply -f <something>.yaml`
    * Helm: `helm upgrade ... tempo ...`
    * Tanka: `tk apply ...`