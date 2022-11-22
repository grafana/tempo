local grafana = import 'grafana.libsonnet';
local prometheus = import 'prometheus.libsonnet';
{
  grafana: grafana,
  prometheus: prometheus,

  fromPrometheusKsonnet(main):: {
    local dashboardFolder = grafana.getFolder(main),
    local mixinRules = prometheus.getMixinRuleNames(main.mixins),

    grafana: grafana.fromMap(main.grafanaDashboards, dashboardFolder)
             + grafana.fromMixins(main.mixins),

    prometheus: prometheus.fromMapsFiltered(main.prometheusAlerts, mixinRules)
                + prometheus.fromMapsFiltered(main.prometheusRules, mixinRules)
                + prometheus.fromMixins(main.mixins),
  },

  resource: (import 'resource.libsonnet'),
  dashboard: grafana.dashboard,
  folder: grafana.folder,
  datasource: grafana.datasource,
  rule_group: prometheus.rule_group,
  synthetic_monitoring_check: (import 'check.libsonnet'),

}
