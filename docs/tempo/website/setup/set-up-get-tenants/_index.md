---
title: Set up GET tenants
weight: 400
---

# Set up a tenant in your Grafana Enterprise Traces cluster

Tenants provide a mechanism for log stream isolation.
Access policies may be set on a per-tenant basis.
Authorization of requests is based on specified access policies.

These instructions assume that you have the [Grafana Enterprise Traces (GET) plugin]({{< relref "../setup-get-pluing-grafana" >}}) installed.
Use this plugin to create tenants, access policies, and tokens for your GET cluster.
The steps include:

1. Create a GET tenant.
1. Create an access policy.
1. Configure remote-write to your tenant.
1. Set up your tenant as a Grafana data source.
1. Visualize your data.

## Create a GET tenant

Once a cluster is running, new tenants may be created.
To create a GET tenant:

1. Within Grafana Enterprise, navigate to **Grafana Enterprise Traces** > **Tenants**.
1. Click on **Create tenant**.

1. Choose a name for this tenant.
For demonstration purposes, use the name `dev-tenant`.

1. Select the cluster.
1. Click **Save**.

## Create an access policy

Access policies are used to authorize actions and operations by specified tenants. 
Access policies have a realm, which defines the set of tenants they apply to, and a scope which defines the set of actions that they confer permissions to use.

Grafana Enterprise requires a data source access policy and token to access the traces in the tenant named `dev-tenant`.

1. Navigate to **Grafana Enterprise Traces** > **Access policies**.
1. Click **Create access policy**.
1. Choose a name for the policy.
For demonstration purposes, use `dev-read-write-policy`.
1. Under **Scopes**, select **Yes** to the question **Are you planning on creating a data source that uses this access policy?** 
1. Enable the scope **traces:read** and **traces:write** to create an access policy that grants access for both reading and writing trace data.
1. Select the tenant `dev-tenant`.
1. Click on **Create**.
1. From the newly created access policy, click **Add token**.
1. Name the token `dev-token` and click on **Create**.
1. In the next window, copy the token by selecting **Copy to clipboard**.

At this point, you can add a data source to your Grafana Enterprise instance by using the **Create a data source** option on this dialog or by manually adding one using the procedure below.
1. Select **Create a data source**. The data source fields are automatically configured based upon your instance. 
2. Verify the created data source by opening the **Configuration** option in the Grafana menu and then selecting **Data sources**.

## Set up remote-write to your tenant

To enable writes to your cluster, add a remote-write configuration snippet to the configuration file of an existing Grafana Agent.
If you do not have an existing traces collector, refer to [Set up with Grafana Agent](https://grafana.com/docs/agent/latest/set-up/).
For Kubernetes, refer to the [Grafana Agent Traces Kubernetes quick start guide](https://grafana.com/docs/grafana-cloud/kubernetes-monitoring/agent-k8s/k8s_agent_traces/).

An example agent configuration would be:

```yaml
traces:
  configs:
  - name: default
    receivers:
      jaeger:
        protocols:
          grpc:           # listens on the default jaeger grpc port: 14250
          thrift_binary:  # 6832
          thrift_compact: # 6831
          thrift_http:    # 14268
    remote_write:
      - endpoint: <get-gateway-host>:<http_listen_port>
        insecure: true  # only add this if TLS is not required
        basic_auth:
          username: dev-tenant
          password: ZGV2LXJlYWQtd3JpdGVyLXBvbGljeS1kZW1vLXRva2VuOjY/ezduMTVhJDQvPGMvLzQ1SzgsJjFbMQ==
    batch:
      timeout: 5s
```

If you have followed the [Tanka GET installation example]({{< relref "../setup/tanka.md" >}}), then the `endpoint` value would be:

```bash
gateway.enterprise-traces.svc.cluster.local:3200.
```

## Create a Grafana data source

To allow Grafana to read traces from GET, you must create a Tempo data source with the proper credentials.

1. Navigate to **Configuration ≫ Data Sources**.

1. Click on **Add new data source**.

1. Select **Tempo**.

1. Set the URL to `http://<get-gateway-host>:<http_listen_port>/<http_api_prefix>`, filling in the path to your gateway and the configured HTTP API prefix. 

1. Enable **Basic Auth**. Use **User** `dev-tenant` and the token from your clipboard as the **Password**.

1. Click **Save & Test**.  

You should see a message that says `Data source is working`. 

If you see an error that says `Data source is not working: failed to get trace with id: 0`, check your Grafana version. 
If your Grafana version is < 8.2.2, this is a bug in Grafana itself and the error can be ignored. 
The underlying data source will still work as expected. 
Upgrade your Grafana to 8.2.2 or later to get the [fix for the error message](https://github.com/grafana/grafana/pull/38018).

## Visualize your data

Once you have created a data source, you can visualize your traces in the **Grafana Explore** page.

### Test your configuration using the TNS application 

You can use The New Stack (TNS) application to test GET data.

1. Navigate to https://github.com/grafana/tns/tree/main/production/k8s-yamls to get the Kubernetes manifests for the TNS application. 
1. Clone the repository using commands similar to the ones below:
    ```bash
      mkdir ~/tmp
      git clone git+ssh://github.com/grafana/tns
      cp tns/production/k8s-yamls/* <targetDir>
    ```
1. Change to the new directory: `cd <targetDir>`
1. In each of the `-dep.yaml` manifests, alter the `JAEGER_AGENT_HOST` to the Grafana Agent location. For example, based on the above Grafana Agent install:
   ```yaml
    	Env:
    	- name: JAEGER_AGENT_HOST
  	  value: grafana-agent-traces.default.svc.cluster.local
   ```
2. Deploy the TNS application. It will deploy into the default namespace.
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
1. Go to Grafana Enterprise and select the **Explore** menu item.
1. Select the **GET data source** from the list of data sources.
1. Copy the trace ID into the **Trace ID** edit field.
1. Select **Run query**. 
1. The trace will be displayed in the traces **Explore** panel.
