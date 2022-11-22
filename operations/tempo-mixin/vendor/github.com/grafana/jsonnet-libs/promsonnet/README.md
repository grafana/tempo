#PromSonnet

`PromSonnet` is intended as a very simple library for creating Prometheus
alerts and rules. It is 'patching friendly', as in, it maintains the
rules internally as a map, which allows users to easily patch alerts and
recording rules after the fact. In contrast, lists require complex
iteration logic.

Take this example:

```
local prom = import 'prom.libsonnet';
local promRuleGroupSet = prom.v1.ruleGroupSet;
local promRuleGroup = prom.v1.ruleGroup;
{
  prometheus_metamon::
    promRuleGroup.new('prometheus_metamon')
    + promRuleGroup.rule.newAlert(
      'PrometheusDown', {
        expr: 'up{job="prometheus"}==0',
        'for': '5m',
        labels: {
          namespace: 'prometheus',
          severity: 'critical',
        },
        annotations: {
        },
      }
    ),

  prometheusAlerts+:
    promRuleGroupSet.new()
    + promRuleGroupSet.addGroup($.prometheus_metamon),
}
```

If we wanted to change the `for` from `5m` to `10m`, we could do this
simply with code such as:

```
{
  prometheusAlerts+: {
    groups_map+:: {
      prometheus_metamon+:: {
        rules+:: {
          PrometheusDown+:: {
            for: '10m',
          },
        },
      },
    },
  },
}
```

We no longer need to iterate over all alerts to do so.

Better than this though, `PromSonnet` provides a helper function, making
this trivial:
```
prometheusAlerts+: prom.v1.patchRule('prometheus_metamon', 'PrometheusDown', { 'for': '10m' }),
```

You can execute either of these examples in the `promsonnet` directory
via:
```
$ jsonnet example.jsonnet
$ # or:
$ jsonnet patch.jsonnet
```
