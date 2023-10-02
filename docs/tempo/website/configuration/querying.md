---
aliases:
- /docs/tempo/v1.2.1/configuration/querying/
title: Querying with Grafana
weight: 3
---

The way Grafana queries Tempo changed from 7.4.x to 7.5.x. This document aims to explain the difference between the two
and help you set up your datasources appropriately.

## Grafana 7.5.x and higher (easy mode)

Grafana 7.5.x and higher can query Tempo directly. Point the Grafana data source at your Tempo query frontend (or single
binary) and enter the URL: `http://<tempo hostname>:<http port number>`. For most of [our examples](https://github.com/grafana/tempo/tree/main/example/docker-compose) the following works.

<p align="center"><img src="../ds75.png" alt="Grafana 7.5.x datasource"></p>

Note that the port of 3200 is a common port used in our examples. Tempo default for http is 80.


## Grafana 7.4.x

Grafana 7.4.x is *not* able to query Tempo directly and requires the tempo-query component as an intermediary. In this case
you need to run Tempo-Query and direct it at Tempo proper. Check out [the Grafana 7.4.x example](https://github.com/grafana/tempo/tree/main/example/docker-compose/grafana7.4) to help with configuration.

The url entered will be `http://<tempo-query hostname>:16686/`.