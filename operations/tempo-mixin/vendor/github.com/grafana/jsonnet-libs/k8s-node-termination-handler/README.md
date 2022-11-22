# k8s-node-termination-handler

The node termination handler is a daemonSet that listens for the termination signal send by the cloud provider, for GKE
that is 30 seconds before termination. The process will delete as many pods as it can before the node is terminated.
This should give the kube-scheduler a head start in rescheduling the workloads before the proclaimed termination of the
node. It reduces the potential downtime of the workloads.

This lib currently covers GKE (gcp provides https://github.com/GoogleCloudPlatform/k8s-node-termination-handler). It
would be nice to add libs for AKS (azure) and ECS (aws provides https://github.com/aws/aws-node-termination-handler/).

## Usage

```
local gke_termination_handler = import 'github.com/jsonnet-libs/k8s-node-termination-handler/gke.libsonnet';

{
  gke_termination_handler: gke_termination_handler + {
    namespace:: 'kube-system',
    slack_webhook:: 'http://hook.slack.com/AAABBBCCC1112222333/',
  },
}
```
