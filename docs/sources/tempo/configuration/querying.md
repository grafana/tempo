---
title: Query Tempo with Grafana
menuTitle: Query Tempo with Grafana
description: Learn how to configure and query Tempo with Grafana.
weight: 40
---

<!-- Page is being deprecated because it describes versions of Grafana that are no longer supported. -->

# Query Tempo with Grafana

Grafana can query Tempo directly. This feature has been enabled since Grafana 7.5.x.

Grafana Cloud comes pre-configured with a Tempo data source.

If you are using Grafana on-prem, you need to [set up the Tempo data source](/docs/grafana/latest/datasources/tempo).

{{% admonition type="tip" %}}
If you want to see what you can do with tracing data in Grafana, try the [Intro to Metrics, Logs, Traces, and Profiling example]({{< relref "../getting-started/docker-example" >}}).
{{% /admonition %}}

This video explains how to add data sources, including Loki, Tempo, and Mimir, to Grafana and Grafana Cloud. Tempo data source set up starts at 4:58 in the video.

{{< youtube id="cqHO0oYW6Ic" >}}

## Configure the data source

To query Tempo with Grafana:

1. Point the Grafana data source at your Tempo query frontend (or monolithic mode Tempo).
1. Enter the URL: `http://<tempo hostname>:<http port number>`. For most of [the Tempo examples](https://github.com/grafana/tempo/tree/main/example/docker-compose) the following works.

The port of 3200 is a common port used in our examples. Tempo default HTTP port is 80.

Prior to Grafana 7.5.x, Grafana was not able to query Tempo directly and required an intermediary, Tempoo-Query.
This [the Grafana 7.4.x example](https://github.com/grafana/tempo/tree/main/example/docker-compose/grafana7.4) to explains  configuration. The URL entered is `http://<tempo-query hostname>:16686/`.
