---
title: Set up a test app for a Tempo cluster
menuTitle: Set up a test application for a Tempo cluster
description: Learn how to set up a test app for your Tempo cluster and visualize data.
weight: 400
---

# Set up a test application for a Tempo cluster

Once you've set up a Grafana Tempo cluster, you need to write some traces to it and then query the traces from within Grafana.

## Before you begin

You'll need:

* Grafana 9.0.0 or higher
* Microservice deployments require the Tempo querier URL, for example: `http://query-frontend.tempo.svc.cluster.local:3200`

Refer to [Deploy Grafana on Kubernetes](https://grafana.com/docs/grafana/latest/setup-grafana/installation/kubernetes/#deploy-grafana-on-kubernetes) if you are using Kubernetes.
Otherwise, refer to [Install Grafana](https://grafana.com/docs/grafana/latest/installation/) for more information.

## Set up remote-write to your Tempo cluster

To enable writes to your cluster, add a remote-write configuration snippet to the configuration file of an existing Grafana Agent.
If you do not have an existing traces collector, refer to [Set up with Grafana Agent](https://grafana.com/docs/agent/latest/set-up/).
For Kubernetes, refer to the [Grafana Agent Traces Kubernetes quick start guide](https://grafana.com/docs/grafana-cloud/kubernetes-monitoring/agent-k8s/k8s_agent_traces/).

The example agent Kubernetes ConfigMap configuration below opens many trace receivers (note that the remote write is onto the Tempo cluster using OTLP gRPC):

```yaml
kind: ConfigMap
metadata:
  name: grafana-agent-traces
apiVersion: v1
data:
  agent.yaml: |
    traces:
        configs:
          - batch:
                send_batch_size: 1000
                timeout: 5s
            name: default
            receivers:
                jaeger:
                    protocols:
                        grpc: null
                        thrift_binary: null
                        thrift_compact: null
                        thrift_http: null
                opencensus: null
                otlp:
                    protocols:
                        grpc: null
                        http: null
                zipkin: null
            remote_write:
              - endpoint: <tempoDistributorServiceEndpoint>
                insecure: true  # only add this if TLS is not required
            scrape_configs:
              - bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
                job_name: kubernetes-pods
                kubernetes_sd_configs:
                  - role: pod
                relabel_configs:
                  - action: replace
                    source_labels:
                      - __meta_kubernetes_namespace
                    target_label: namespace
                  - action: replace
                    source_labels:
                      - __meta_kubernetes_pod_name
                    target_label: pod
                  - action: replace
                    source_labels:
                      - __meta_kubernetes_pod_container_name
                    target_label: container
                tls_config:
                    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
                    insecure_skip_verify: false
```

If you have followed the [Tanka Tempo installation example]({{< relref "../setup/tanka" >}}), then the `endpoint` value would be:

```bash
distributor.tempo.svc.cluster.local:4317
```

Apply the ConfigMap with:

```bash
kubectl apply --namespace default -f agent.yaml
```

Deploy Grafana Agent using the instructions from the relevant instructions above.

## Create a Grafana Tempo data source

To allow Grafana to read traces from Tempo, you must create a Tempo data source.

1. Navigate to **Configuration ≫ Data Sources**.

2. Click on **Add data source**.

3. Select **Tempo**.

4. Set the URL to `http://<tempo-host>:<http_listen_port>/`, filling in the path to your gateway and the configured HTTP API prefix. If you have followed the [Tanka Tempo installation example]({{< relref "../setup/tanka.md" >}}), this will be: `http://query-frontend.tempo.svc.cluster.local:3200/`

5. Click **Save & Test**.

You should see a message that says `Data source is working`.

If you see an error that says `Data source is not working: failed to get trace with id: 0`, check your Grafana version.
If your Grafana version is < 8.2.2, this is a bug in Grafana itself and the error can be ignored.
The underlying data source will still work as expected.
Upgrade your Grafana to 8.2.2 or later to get the [fix for the error message](https://github.com/grafana/grafana/pull/38018).

## Visualize your data

Once you have created a data source, you can visualize your traces in the **Grafana Explore** page.
For more information, refer to [Tempo in Grafana]({{< relref "../getting-started/tempo-in-grafana" >}}).

### Test your configuration using the TNS application

You can use The New Stack (TNS) application to test Tempo data.

1. Create a new directory to store the TNS manifests.
1. Navigate to `https://github.com/grafana/tns/tree/main/production/k8s-yamls` to get the Kubernetes manifests for the TNS application.
1. Clone the repository using commands similar to the ones below (where `<targetDir>` is the directory you used to store the manifests):

    ```bash
      mkdir ~/tmp
      cd ~/tmp
      git clone git+ssh://github.com/grafana/tns
      cp tns/production/k8s-yamls/* <targetDir>
    ```

1. Change to the new directory: `cd <targetDir>` .
1. In each of the `-dep.yaml` manifests, alter the `JAEGER_AGENT_HOST` to the Grafana Agent location. For example, based on the above Grafana Agent install:
   ```yaml
   env:
   - name: JAEGER_AGENT_HOST
     value: grafana-agent-traces.default.svc.cluster.local
   ```
1. Deploy the TNS application. It will deploy into the default namespace.
   ```bash
	 kubectl apply -f app-svc.yaml,db-svc.yaml,loadgen-svc.yaml,app-dep.yaml,db-dep.yaml,loadgen-dep.yaml
   ```
1. Once the application is running, look at the logs for one of the pods (such as the App pod) and find a relevant trace ID. For example:
   ```bash
  	kubectl logs $(kubectl get pod -l name=app -o jsonpath="{.items[0].metadata.name}")
    level=debug traceID=50075ac8b434e8f7 msg="GET / (200) 1.950625ms"
    level=info msg="HTTP client success" status=200 url=http://db duration=1.297806ms traceID=2c2fd669c388e76
    level=debug traceID=2c2fd669c388e76 msg="GET / (200) 1.70755ms"
    level=info msg="HTTP client success" status=200 url=http://db duration=1.853271ms traceID=79058bb9cc39acfb
    level=debug traceID=79058bb9cc39acfb msg="GET / (200) 2.300922ms"
    level=info msg="HTTP client success" status=200 url=http://db duration=1.381894ms traceID=7b0e0526f5958549
    level=debug traceID=7b0e0526f5958549 msg="GET / (200) 2.105263ms"
   ```
1. Go to Grafana and select the **Explore** menu item.
1. Select the **Tempo data source** from the list of data sources.
1. Copy the trace ID into the **Trace ID** edit field.
1. Select **Run query**.
1. Confirm that the trace is displayed in the traces **Explore** panel.
