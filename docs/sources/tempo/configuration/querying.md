---
title: Use Tempo with Grafana
menuTitle: Use Tempo with Grafana
description: Learn how to configure and query Tempo with Grafana.
weight: 900
---

<!-- Page is being deprecated because it describes versions of Grafana that are no longer supported. -->

# Use Tempo with Grafana

You can use Tempo as a data source in Grafana to Tempo can query Grafana directly. Grafana Cloud comes pre-configured with a Tempo data source.

If you are using Grafana on-prem, you need to [set up the Tempo data source](/docs/grafana/<GRAFANA_VERSION>/datasources/tempo).

{{% admonition type="tip" %}}
If you want to see what you can do with tracing data in Grafana, try the [Intro to Metrics, Logs, Traces, and Profiling example]({{< relref "../getting-started/docker-example" >}}).
{{% /admonition %}}

This video explains how to add data sources, including Loki, Tempo, and Mimir, to Grafana and Grafana Cloud. Tempo data source set up starts at 4:58 in the video.

{{< youtube id="cqHO0oYW6Ic" >}}

## Configure the data source

For detailed instructions on the Tempo dta source in Grafana, refer to [Tempo data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/).

To configure Tempo with Grafana:

1. Point the Grafana data source at your Tempo query frontend (or monolithic mode Tempo).
1. Enter the URL: `http://<tempo hostname>:<http port number>`. For most of [the Tempo examples](https://github.com/grafana/tempo/tree/main/example/docker-compose) the following works.

The port of 3200 is a common port used in our examples. Tempo default HTTP port is 80.

## Query the data source

Refer to [Tempo in Grafana]({{< relref "../getting-started/tempo-in-grafana" >}}) for an overview about how tracing data can be viewed and used in Grafana.

For information on querying the Tempo data source, refer to [Tempo query editor](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/query-editor/).