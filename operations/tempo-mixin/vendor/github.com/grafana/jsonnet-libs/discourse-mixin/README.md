# Discourse Mixin

The [Discourse](https://meta.discourse.org/) mixin contains a variety of configurable Grafana dashboards and alerts that are sourced from the [Discourse Prometheus Metrics Plugin](https://github.com/discourse/discourse-prometheus).

The Discourse mixin contains the following dashboards:

- Discourse Overview
- Discourse Jobs Processing

The Discourse mixin contains the following alerts:

- DiscourseRequestsHigh5xxErrors
- DiscourseRequestsHigh4xxErrors

## Discourse Overview

The Discourse Overview dashboard highlights web traffic, underlying rails controller metrics, request activity, and pageviews. It also outlines some useful statistics surrounding latest median request latency for all the controllers that publish these.

![First screenshot of the overview dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/discourse/screenshots/discourse_overview_1.png)
![Second screenshot of the overview dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/discourse/screenshots/discourse_overview_2.png)

## Discourse Jobs Processing

The Discourse Jobs Processing dashboard displays information about job duration and browser based memory usage. It also displays information about worker processes like `Sidekiq` and `Web` workers.

![Screenshot of the jobs processing dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/discourse/screenshots/discourse_jobs_1.png)

## Alerts Overview

| Alert                     | Description                                                                                                                                        |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| DiscourseRequestsHigh5xxErrors | The Discourse environment is responding with a lot of 5XXs. Could be indicative that the instance is experiencing serious problems.                |
| DiscourseRequestsHigh4xxErrors | The Discourse environment exceeded the percentage of requests that return 4XX response codes. Could be indicative of improper use of the Instance. |

## Generating dashboards and alerts

```bash
make
```

Creates a generated `dashboards_out` directory and `prometheus_alerts.yaml` that can be imported into Grafana.

For more advanced uses of mixins, see [mixin documentation.](
https://github.com/monitoring-mixins/docs)
