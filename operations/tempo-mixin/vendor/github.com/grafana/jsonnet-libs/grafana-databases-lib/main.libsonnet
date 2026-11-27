// grafana-databases-lib: dashboard panels and rows shared by the database mixins. See README.md.
local autoscaling = import './autoscaling.libsonnet';
local dashboard = import './dashboard.libsonnet';

{
  // new returns a library instance. `config` is merged into the library-wide
  // config like withConfigMixin; each area has its own config mixin:
  //
  //   local databases =
  //     dblib.new({ clusterLabel: 'cluster' })
  //     + dblib.autoscaling.withConfigMixin({ components: { querier: { hpaName: 'keda-hpa-querier' } } });
  //
  //   databases.autoscaling.panels.querier.replicas
  //   databases.autoscaling.rows.querier
  //
  // Panels and rows are dashboard v2 elements and rows: assemble them with
  // elements() and layout(), or convert them with legacy for grafana-builder.
  new(config={}):: {
    local this = self,

    // Areas read `config` through `self`, so a config mixin added after new()
    // still reaches every panel and row.
    config: {
      local shared = self,
      // Dashboard datasource variable, or a datasource uid.
      datasource: '$datasource',
      clusterLabel: 'cluster',
      namespaceLabel: 'namespace',
      // Matcher applied to kube-state-metrics series.
      selector: '%s=~"$cluster", %s=~"$namespace"' % [shared.clusterLabel, shared.namespaceLabel],
      // Labels used to join series.
      groupBy: [shared.clusterLabel, shared.namespaceLabel],

      autoscaling: autoscaling.defaults(shared),
    },

    autoscaling: autoscaling.new(this.config),
  } + self.withConfigMixin(config),

  // withConfigMixin merges into the library-wide config.
  withConfigMixin(config):: { config+: config },

  autoscaling:: {
    // withConfigMixin merges into config.autoscaling.
    withConfigMixin(config):: { config+: { autoscaling+: config } },
    // withPanelMixin merges `patch` into the named panel
    // autoscaling.panels.<component>.<panel>, and so into the rows using it.
    withPanelMixin(component, panel, patch):: {
      autoscaling+: { panels+: { [component]+: { [panel]+: patch } } },
    },
  },

  elements:: dashboard.elements,
  layout:: dashboard.layout,
  row:: dashboard.row,
  panel:: dashboard.panel,
  query:: dashboard.query,
  withViz:: dashboard.withViz,

  legacy:: import './legacy.libsonnet',
}
