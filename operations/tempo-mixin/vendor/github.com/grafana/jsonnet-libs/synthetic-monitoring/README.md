# Synthetic Monitoring Jsonnet Library

This library facilitates the configuration of checks for Grafana Synthetic
Monitoring. It is intended to be used alongside [Grizzly](https://github.com/grafana/grizzly).

Grafana Labs' Synthetic Monitoring is a blackbox monitoring solution provided
as a part of Grafana Cloud. It provides users with insights into how their
applications and services are behaving from an external point of view.

## Usage

This library is designed to be used with [Grizzly](https://github.com/grafana/grizzly),
which supports the Synthetic Monitoring API.

Here is an example usage:
**`main.jsonnet:`**
```
local sm = import 'synthetic-monitoring/sm.libsonnet';
  
{
  syntheticMonitoring+:: {
    grafanaHttpCheck: sm.http.new('grafana', 'https://grafana.com/')
                      + sm.withProbes('all'),  // enable all probes
    grafanaPingCheck: sm.ping.new('grafana', 'grafana.com')
                      + sm.withProbes('continents'),  // one check per continent
    grafanaDnsCheck: sm.dns.new('grafana-dns', 'grafana.com') // Combination of Job name and Target must be unique
                     + sm.withProbes('europe'),  // just check from Europe
    grafanaTcpCheck: sm.tcp.new('grafana', 'grafana.com:443')
                     + sm.withProbes('small'),  // just use a smaller, predefined set of checks
  },
}
```

To apply this to your cluster, set the `GRAFANA_SM_TOKEN` envvar to an API key from your
Grafana.com account, then execute:

```
$ grr apply main.jsonnet
```

This should create four probes for you.
