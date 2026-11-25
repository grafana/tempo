// autoscaling: HPA and KEDA autoscaling panels and rows. See README.md.
local dashboard = import './dashboard.libsonnet';
local g = import './g.libsonnet';

local ts = g.panel.timeSeries;

{
  // defaults returns the default config.autoscaling, given the library-wide
  // config it lives in.
  defaults(shared):: {
    // Prefix KEDA adds to the ScaledObject name when creating its HPA.
    hpaPrefix: 'keda-hpa-',
    // Unnamed KEDA triggers have no `scaler` label, only `metric`
    // (s0-prometheus, ...), so the default legend relies on `metric`. Set
    // '{{ scaler }}' when the ScaledObjects name their triggers.
    triggerLegend: '{{ scaledObject }} {{ metric }}',
    // null lets Grafana fit the Y axis to the data.
    axisMin: null,
    // Shown when a component has no HPA. null shows "No data".
    noValue: 'Not auto-scaled',
    // KEDA exports the scaled object's namespace as `exported_namespace`, as
    // `namespace` is the KEDA operator's own.
    kedaSelector: '%s=~"$cluster", exported_namespace=~"$namespace"' % shared.clusterLabel,
    // Component key -> { hpaName, title, keda, scalers, extraReplicaQueries }; see README.md.
    components: {},
  },

  // new builds the autoscaling area from the library-wide config.
  new(config):: {
    local area = self,
    local a = config.autoscaling,
    local groupBy = std.join(', ', config.groupBy),
    local params = {
      selector: config.selector,
      kedaSelector: a.kedaSelector,
      hpaPrefix: a.hpaPrefix,
      groupBy: groupBy,
    },

    local component(key) =
      if std.objectHas(a.components, key) then a.components[key]
      else error 'autoscaling: unknown component "%s"' % key,

    local query(expr, legend) =
      dashboard.query('prometheus', config.datasource, { expr: expr, legendFormat: legend }),

    // v2 vizConfig holds the same options and fieldConfig as a classic panel,
    // so grafonnet's typed timeSeries helpers still build them.
    local base(name, title, description, queries, viz={}) =
      dashboard.panel(
        name,
        'timeseries',
        title,
        description,
        queries,
        viz=ts.standardOptions.withUnit('short')
            + (if a.axisMin != null then ts.standardOptions.withMin(a.axisMin) else {})
            + (if a.noValue != null then ts.standardOptions.withNoValue(a.noValue) else {})
            + ts.fieldConfig.defaults.custom.withFillOpacity(1)
            + ts.fieldConfig.defaults.custom.withLineWidth(1)
            + ts.fieldConfig.defaults.custom.withShowPoints('never')
            + ts.options.legend.withShowLegend(true)
            + ts.options.tooltip.withMode('multi')
            + viz,
        version=ts.new('').pluginVersion,
      ),

    // Only kube_horizontalpodautoscaler_info carries the scaled workload's name,
    // so it is joined in for the legend.
    local hpaQuery(metric, hpaName) = |||
      max by (scaletargetref_name) (
        %(metric)s{%(selector)s, horizontalpodautoscaler=~"%(hpa)s"}
        * on (%(groupBy)s, horizontalpodautoscaler) group_left (scaletargetref_name)
        group by (%(groupBy)s, horizontalpodautoscaler, scaletargetref_name) (kube_horizontalpodautoscaler_info{%(selector)s, horizontalpodautoscaler=~"%(hpa)s"})
      )
    ||| % (params { metric: metric, hpa: hpaName }),

    // `extraQueries` is a list of { expr, legend } added as series (e.g. a ReplicaTemplate).
    replicasPanel(componentKey, extraQueries=[], name='autoscaling-%s-replicas' % componentKey)::
      local hpaName = component(componentKey).hpaName;
      local line(name, dashed) = ts.standardOptions.withOverridesMixin([
        ts.standardOptions.override.byRegexp.new('/%s .+/' % name)
        + ts.standardOptions.override.byRegexp.withPropertiesFromOptions(
          ts.fieldConfig.defaults.custom.withFillOpacity(0)
          + (if dashed then ts.fieldConfig.defaults.custom.withLineStyle({ fill: 'dash' }) else {})
        ),
      ]);
      base(
        name,
        'Replicas',
        'The minimum and maximum replicas the HPA may use, and the current and desired replicas it reports. A component at its maximum cannot scale further.',
        [
          query(hpaQuery('kube_horizontalpodautoscaler_spec_min_replicas', hpaName), 'Min {{ scaletargetref_name }}'),
          query(hpaQuery('kube_horizontalpodautoscaler_status_current_replicas', hpaName), 'Current {{ scaletargetref_name }}'),
          query(hpaQuery('kube_horizontalpodautoscaler_status_desired_replicas', hpaName), 'Desired {{ scaletargetref_name }}'),
          query(hpaQuery('kube_horizontalpodautoscaler_spec_max_replicas', hpaName), 'Max {{ scaletargetref_name }}'),
        ] + [query(q.expr, q.legend) for q in extraQueries],
        viz=line('Min', true) + line('Current', false) + line('Desired', false) + line('Max', true),
      ),

    // AverageValue triggers want value / target replicas; Value triggers want
    // value / target * current replicas. Grouped by metric as well as scaler,
    // as triggers are not always named. `scaler` is a regex on the scaler name.
    desiredReplicasPanel(componentKey, scaler='', title='Desired replicas', name='autoscaling-%s-desired-replicas' % componentKey)::
      local p = params {
        hpa: component(componentKey).hpaName,
        kedaSelector+: if scaler != '' then ', scaler=~"%s"' % scaler else '',
      };
      local desired(targetType, multiplier) = |||
        sum by (scaledObject, metric, scaler) (
          # Using `max by ()` so that series churn doesn't break the promQL join
          max by (%(groupBy)s, scaledObject, metric, scaler) (
            label_replace(
              keda_scaler_metrics_value{%(kedaSelector)s},
              "namespace", "$1", "exported_namespace", "(.*)"
            )
          )
          / on (%(groupBy)s, scaledObject, metric) group_left
          max by (%(groupBy)s, scaledObject, metric) (
            label_replace(
              label_replace(
                kube_horizontalpodautoscaler_spec_target_metric{%(selector)s, horizontalpodautoscaler=~"%(hpa)s", metric_target_type="%(targetType)s"},
                "metric", "$1", "metric_name", "(.+)"
              ),
              "scaledObject", "$1", "horizontalpodautoscaler", "%(hpaPrefix)s(.*)"
            )
          )%(multiplier)s
        )
      ||| % (p { targetType: targetType, multiplier: multiplier });
      local timesCurrent = |||

        * on (%(groupBy)s, scaledObject) group_left
        max by (%(groupBy)s, scaledObject) (
          label_replace(
            kube_horizontalpodautoscaler_status_current_replicas{%(selector)s, horizontalpodautoscaler=~"%(hpa)s"},
            "scaledObject", "$1", "horizontalpodautoscaler", "%(hpaPrefix)s(.*)"
          )
        )
      ||| % p;
      base(
        name,
        title,
        'The replicas each autoscaling trigger asks for: the scaling metric exposed by KEDA divided by its target. Ignores the min/max constraints the HPA applies later.',
        [query('(\n%s)\nor\n(\n%s)' % [desired('average', ''), desired('value', timesCurrent)], a.triggerLegend)]
      ),

    // KEDA 2.16 renamed keda_scaler_errors to keda_scaler_detail_errors_total,
    // so match both. The `* 0` HPA join only restricts the series to the component.
    failuresPanel(componentKey, name='autoscaling-%s-failures' % componentKey)::
      local p = params { hpa: component(componentKey).hpaName };
      base(
        name,
        'Autoscaler failures rate',
        'The rate of failures in the KEDA custom metrics API server. On an error KEDA cannot read the scaling metric, so the autoscaler will not work properly.',
        [query(|||
          sum by (scaledObject, metric, scaler) (
            # Using `max by ()` so duplicate series for the same trigger aren't counted twice
            max by (%(groupBy)s, scaledObject, metric, scaler) (
              label_replace(
                rate({__name__=~"keda_scaler_detail_errors_total|keda_scaler_errors", %(kedaSelector)s}[$__rate_interval]),
                "namespace", "$1", "exported_namespace", "(.+)"
              )
            )
            + on (%(groupBy)s, scaledObject, metric) group_left
            max by (%(groupBy)s, scaledObject, metric) (
              label_replace(
                label_replace(
                  kube_horizontalpodautoscaler_spec_target_metric{%(selector)s, horizontalpodautoscaler=~"%(hpa)s"} * 0,
                  "metric", "$1", "metric_name", "(.+)"
                ),
                "scaledObject", "$1", "horizontalpodautoscaler", "%(hpaPrefix)s(.*)"
              )
            )
          )
        ||| % p, '%s failures' % a.triggerLegend)]
      ),

    panels: {
      [name]: {
        local comp = a.components[name],
        replicas: area.replicasPanel(name, std.get(comp, 'extraReplicaQueries', [])),
        desiredReplicas: area.desiredReplicasPanel(name),
        desiredReplicasByScaler: {
          [scaler]:
            local s = comp.scalers[scaler];
            local scalerTitle = std.get(s, 'title', scaler);
            area.desiredReplicasPanel(
              name,
              scaler=std.get(s, 'match', '.*%s.*' % scaler),
              title='Scaling metric (%s): Desired replicas' % scalerTitle,
              name='autoscaling-%s-desired-replicas-%s' % [name, scaler],
            )
            + (if a.noValue != null then dashboard.withViz(ts.standardOptions.withNoValue('Not scaled on %s' % scalerTitle)) else {})
          for scaler in std.objectFields(std.get(comp, 'scalers', {}))
        },
        failures: area.failuresPanel(name),
      }
      for name in std.objectFields(a.components)
    },

    // Rows take their panels from `area.panels` (self) rather than building
    // them, so overriding a named panel also changes it in its row.
    rows: {
      [name]:
        local comp = a.components[name];
        local panels = area.panels[name];
        dashboard.row(
          '%s - autoscaling' % std.get(comp, 'title', name),
          [panels.replicas]
          + if std.get(comp, 'keda', true) then
            (
              if std.length(std.get(comp, 'scalers', {})) > 0
              then std.objectValues(panels.desiredReplicasByScaler)
              else [panels.desiredReplicas]
            ) + [panels.failures]
          else []
        )
      for name in std.objectFields(a.components)
    },
  },
}
