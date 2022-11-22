# Home Assistant Mixin
A general purpose dashboard built around the metrics exposed by the Home assistant in-built [Prometheus exporter](https://www.home-assistant.io/integrations/prometheus/).

## Requirements
This mixin makes heavy use of querying with the `__name__` matcher, which requires that the underlying Prometheus-compatible TSDB is storing data using blocks, rather than chunks. The mixin is written to target Hosted Grafana, which is deployed in combination with a compatible cortex instance.

## Features
### Custom Namespace
The mixin supports a custom namespace for metrics if defined in the [exporter configuration](https://www.home-assistant.io/integrations/prometheus/#namespace).

The user must type their custom namespace into the `prefix` variable of the dashboard

### Inactive Entity Filtering
By default, the mixin will show inactive entities as greyed out tiles. Inactive entities can be removed from the dashboard by selecting "Exclude" from the `inactive` variable of the dashboard
