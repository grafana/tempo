---
headless: true
description: Shared file for metamonitoring GET and Tempo.
labels:
  products:
    - enterprise
    - oss
---

[//]: # 'This file documents the best practices for monitoring for Tempo.'
[//]: # 'This shared file is included in these locations:'
[//]: # '/tempo/docs/sources/tempo/operations/monitor/set-up-monitoring.md'
[//]: #
[//]: # 'If you make changes to this file, verify that the meaning and content are not changed in any place where the file is included.'
[//]: # 'Any links should be fully qualified and not relative: /docs/grafana/ instead of ../grafana/.'


Metamonitoring for Tempo is handled by the [Grafana Kubernetes Helm chart](https://github.com/grafana/k8s-monitoring-helm) (>=v2.1). Metamonitoring can be used with both microservices and single binary deployments of Tempo.

The Helm chart configures Grafana Alloy to collect metrics and logs.

## Steps

This procudure uses the [Grafana Kubernetes Helm chart](https://github.com/grafana/k8s-monitoring-helm) and the `values.yml` file sets parameters in the Helm chart.

1. Add the Grafana Helm Chart repository, or update, if already added. 

    ```
    helm repo add grafana https://grafana.github.io/helm-charts  
    helm repo update
    ```

1. Create a new file named `values.yml`. Add the following example into your `values.yml` file and save it. Where indicated, add the values specific to your instance.

    ```yaml
    cluster:
        name: traces # Name of the cluster. This populates the cluster label.

    integrations:
        tempo:
            instances:
              - name: "traces" # This is the name for the instance label that reports.
                namespaces:
                    - traces # This is the namespace that is searched for Tempo instances. Change this accordingly.
                metrics:
                    enabled: true
                    portName: prom-metrics
                logs:
                    enabled: true
                labelSelectors:
                    app.kubernetes.io/name: tempo

    alloy:
        name: "traces-monitoring"

    destinations:
    - name: "metrics"
      type: prometheus
      url: "<url>" # URL for Prometheus. Should look similar to "https://<prometheus host>/api/prom/push".
      auth:
        type: basic
        username: "<username>"
        password: "<password>"

    - name: "logs"
      type: loki
      url: "<url>" # URL for Loki. Should look similar to "https://<loki host>/loki/api/v1/push".
      auth:
        type: basic
        username: "<username>" 
        password: "<password>"

    alloy-metrics:
        enabled: true

    podLogs:
        enabled: true
        gatherMethod: kubernetesApi
        namespaces: [traces] # Set to namespace from above under instances.
        collector: alloy-singleton

    alloy-singleton:
        enabled: true

    alloy-metrics:
        enabled: true # Sends Grafana Alloy metrics to ensure the monitoring is working properly.
    ```

1. Install the Helm chart using the following command to create Grafana Alloy instances to scrape metrics and logs:

    ```
    helm install k8s-monitoring grafana/k8s-monitoring \
    --namespace monitoring \
    --create-namespace \
    -f values.yml
    ```

1. Verify that data is being sent to Grafana. 
    - Log into Grafana. 
    - Select Metrics Drilldown and select `cluster=<cluster.name>` where `cluster.name` is the name specified in the `values.yml` file. 
    - Do the same for Logs Drilldown.

    This example doesn’t include ingestion for any other data such as traces for sending to Tempo, but can be included with some configuration updates.
    Refer to [Configure Alloy to remote-write to Tempo](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/set-up-test-app/) for more information.

## Install Tempo dashboards in Grafana

Alloy scrapes metrics from Tempo and sends them to Mimir or another Prometheus compatible time-series database.
You can then monitor Tempo using the mixins.

Tempo ships with mixins that includes:

* Relevant dashboards for overseeing the health of Tempo as a whole, as well as its individual components
* Recording rules that simplify the generation of metrics for dashboards and free-form queries
* Alerts that trigger when Tempo falls out of operational parameters

To install the mixins in Grafana, you need to:

1. Download the mixin dashboards from the Tempo repository.

1. Import the dashboards in your Grafana instance.

1. Upload `alerts.yaml` and `rules.yaml` files for Mimir or Prometheus

### Download the `tempo-mixin` dashboards

1. First, clone the Tempo repository from Github:
   ```bash
   git clone git+ssh://github.com/grafana/tempo
   ```

1. Once you have a local copy of the repository, navigate to the `operations/tempo-mixin-compiled` directory.
   ```bash
   cd operations/tempo-mixin-compiled
   ```

This contains a compiled version of the alert and recording rules, as well as the dashboards.

{{< admonition type="note" >}}
If you want to change any of the mixins, make your updates in the `operations/tempo-mixin` directory.
Use the instructions in the [README](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin) in that directory to regenerate the files.
The mixins are generated in the `operations/tempo-mixin-compiled` directory.
{{< /admonition >}}

### Import the dashboards to Grafana

The `dashboards` directory includes the six monitoring dashboards that can be installed into your Grafana instance.
Refer to [Import a dashboard ](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/import-dashboards/)in the Grafana documentation.

{{< admonition type="tip" >}}
Install all six dashboards.
You can only import one dashboard at a time.
Create a new folder in the Dashboards area, for example “Tempo Monitoring”, as an easy location to save the imported dashboards.
{{< /admonition >}}

To create a folder:

1. Open your Grafana instance and select **Dashboards**.
1. Select **New** in the right corner.
1. Select **New folder** from the **New** drop-down.
1. Name your folder, for example, “Tempo Monitoring”.
1. Select **Create**.

To import a dashboard:

1. Open your Grafana instance and select **Dashboards**.
1. Select **New** in the right corner.
1. Select **Import**.
1. On the **Import dashboard** screen, select **Upload.**
1. Browse to `operations/tempo-mixin-compiled/dashboards` and select the dashboard to import.
1. Drag the dashboard file, for example, `tempo-operational.json`, onto the **Upload** area of the **Import dashboard** screen. Alternatively, you can browse to and select a file.
1. Select a folder in the **Folder** drop-down where you want to save the imported dashboard. For example, select Tempo Monitoring created in the earlier steps.
1. Select **Import**.

The imported files are listed in the Tempo Monitoring dashboard folder.

To view the dashboards in Grafana:

1. Select Dashboards in your Grafana instance.
1. Select Tempo Monitoring, or the folder where you uploaded the imported dashboards.
1. Select any files in the folder to view it.

The ‘Tempo Operational’ dashboard shows read (query) information:

![Tempo Operational dashboard](/media/docs/tempo/screenshot-tempo-ops-dashboard.png "Tempo Operational dashboard")

### Add alerts and rules to Prometheus or Mimir

The rules and alerts need to be installed into your Mimir or Prometheus instance.
To do this in Prometheus, refer to the [recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) and [alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/) documentation.

For Mimir, you can use `[mimirtool](https://grafana.com/docs/mimir/latest/manage/tools/mimirtool/)` to upload [rule](https://grafana.com/docs/mimir/latest/manage/tools/mimirtool/#rules) and [alert](https://grafana.com/docs/mimir/latest/manage/tools/mimirtool/#alertmanager) configuration.
Using a default installation of Mimir used as the metrics store for the Alloy configuration, you might run the following:

```bash
mimirtool rules load operations/tempo-mixin-compiles/rules.yml --address=https://mimir-cluster.distributor.mimir.svc.cluster.local:9001

mimirtool alertmanager load operations/tempo-mixin-compiles/alerts.yml --address=https://mimir-cluster.distributor.mimir.svc.cluster.local:9001
```

For Grafana Cloud, you need to add the username and API key as well.
Refer to the [mimirtool](https://grafana.com/docs/mimir/latest/manage/tools/mimirtool/) documentation for more information.
