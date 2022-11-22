local prom = import 'prom.libsonnet';
local promRuleGroupSet = prom.v1.ruleGroupSet;
local promRuleGroup = prom.v1.ruleGroup;
{
  prometheus_metamon::
    promRuleGroup.new('prometheus_metamon')
    + promRuleGroup.rule.newAlert(
      'PrometheusDown', {
        expr: 'up{job="prometheus"} == 0',
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
