local grafana = import 'grafana-builder/grafana.libsonnet';
local utils = import 'mixin-utils/utils.libsonnet';

grafana {
  // Override the dashboard constructor to add:
  // - default tags,
  // - some links that propagate the selected cluster.
  // - disable auto refresh every 10 seconds
  dashboard(title)::
    super.dashboard(title) + {
      // addRowIf adds the row only when the condition holds.
      addRowIf(condition, row):: if condition then self.addRow(row) else self,

      addClusterSelectorTemplates()::
        local d = self {
          refresh: '',
          tags: ['tempo'],
          links: [
            {
              asDropdown: true,
              icon: 'external link',
              includeVars: true,
              keepTime: true,
              tags: ['tempo'],
              targetBlank: false,
              title: 'Tempo Dashboards',
              type: 'dashboards',
            },
          ],
        };

        d.addMultiTemplate('cluster', 'tempo_build_info', $._config.per_cluster_label, allValue=null)
        .addMultiTemplate('namespace', 'tempo_build_info{' + $._config.per_cluster_label + "=~'$cluster'}", 'namespace', allValue=null),
    },

  jobMatcher(job)::
    $._config.per_cluster_label + '=~"$cluster", job=~"($namespace)' + $._config.namespace_selector_separator + '%s"' % job,

  podMatcher(pod)::
    $._config.per_cluster_label + '=~"$cluster", pod=~"' + '%s"' % pod,

  containerMatcher(container)::
    $._config.per_cluster_label + '=~"$cluster", container="' + '%s"' % container,

  queryPanel(queries, legends, legendLink=null)::
    super.queryPanel(queries, legends, legendLink) + {
      targets: [
        target {
          interval: '1m',
        }
        for target in super.targets
      ],
    },

  panelDescription(title, description):: {
    description: |||
      ### %s
      %s
    ||| % [title, description],
  },


  // hiddenLegendQueryPanel is a standard query panel designed to handle a large number of series.  it hides the legend, doesn't fill the series and
  //  sorts the tooltip descending
  hiddenLegendQueryPanel(queries, legends, legendLink=null)::
    $.queryPanel(queries, legends, legendLink) +
    {
      legend: { show: false },
      fill: 0,
      tooltip: { sort: 2 },
    },

  qpsPanel(selector)::
    super.qpsPanel(selector) + {
      targets: [
        target {
          interval: '1m',
        }
        for target in super.targets
      ],
      fieldConfig+: {
        defaults+: {
          unit: 'ops',
        },
      },
    },

  panel(title):: $.timeseriesPanel(title),

  timeseriesPanel(title):: super.timeseriesPanel(title),

  // fork of grafana latency panel with additional_grouping added
  latencyPanel(metricName, selector, multiplier='1e3', additional_grouping=''):: {
    nullPointMode: 'null as zero',
    targets: [
      {
        expr: 'histogram_quantile(0.99, sum(rate(%s_bucket%s[$__rate_interval])) by (le,%s)) * %s' % [metricName, selector, additional_grouping, multiplier],
        format: 'time_series',
        intervalFactor: 2,
        legendFormat: '{{route}} 99th',
        refId: 'A',
        step: 10,
        interval: '1m',
      },
      {
        expr: 'histogram_quantile(0.50, sum(rate(%s_bucket%s[$__rate_interval])) by (le,%s)) * %s' % [metricName, selector, additional_grouping, multiplier],
        format: 'time_series',
        intervalFactor: 2,
        legendFormat: '{{route}} 50th',
        refId: 'B',
        step: 10,
        interval: '1m',
      },
      {
        expr: 'sum(rate(%s_sum%s[$__rate_interval])) by (%s) * %s / sum(rate(%s_count%s[$__rate_interval])) by (%s)' % [metricName, selector, additional_grouping, multiplier, metricName, selector, additional_grouping],
        format: 'time_series',
        intervalFactor: 2,
        legendFormat: '{{route}} Average',
        refId: 'C',
        step: 10,
        interval: '1m',
      },
    ],
    fieldConfig+: {
      defaults+: {
        unit: 'ms',
      },
    },
    yaxes: $.yaxes('ms'),
  },

  //
  // HPA/KEDA autoscaling rows for the Writes and Reads dashboards.
  //

  // autoscalingReplicasPanel shows the HPA min / max bounds next to the
  // current and desired replicas of the component's HPA.
  autoscalingReplicasPanel(component)::
    local hpaQuery(metric) = |||
      max by (scaletargetref_name) (
        %(metric)s{%(ns)s, horizontalpodautoscaler=~"%(hpa_name)s"}
        * on (%(group_by)s, horizontalpodautoscaler) group_left (scaletargetref_name)
        group by (%(group_by)s, horizontalpodautoscaler, scaletargetref_name) (kube_horizontalpodautoscaler_info{%(ns)s, horizontalpodautoscaler=~"%(hpa_name)s"})
      )
    ||| % {
      metric: metric,
      ns: $.namespaceMatcher(),
      group_by: $._config.group_by_cluster,
      hpa_name: $._config.autoscaling[component].hpa_name,
    };

    $.panel('Replicas') +
    { fieldConfig+: { defaults+: { unit: 'short' } } } +
    $.queryPanel(
      [
        hpaQuery('kube_horizontalpodautoscaler_spec_min_replicas'),
        hpaQuery('kube_horizontalpodautoscaler_status_current_replicas'),
        hpaQuery('kube_horizontalpodautoscaler_status_desired_replicas'),
        hpaQuery('kube_horizontalpodautoscaler_spec_max_replicas'),
      ],
      ['min {{scaletargetref_name}}', 'current {{scaletargetref_name}}', 'desired {{scaletargetref_name}}', 'max {{scaletargetref_name}}'],
    ) +
    $.panelDescription(
      'Replicas',
      |||
        The minimum and maximum replicas the HPA is allowed to use, and the current and desired replicas it reports.
        A component sitting at its maximum can't scale further.
      |||
    ),

  // autoscalingDesiredReplicasPanel shows the replicas each trigger asks for,
  // before the HPA min/max bounds are applied. Tempo's triggers are unnamed, so
  // KEDA reports them all as the same scaler; group by ScaledObject and trigger
  // metric (s0-prometheus, s1-cpu, ...) instead. AverageValue triggers want
  // value / target replicas; Value triggers want value / target * current replicas.
  autoscalingDesiredReplicasPanel(component)::
    local params = {
      ns: $.namespaceMatcher(),
      cluster_label: $._config.per_cluster_label,
      group_by: $._config.group_by_cluster,
      hpa_name: $._config.autoscaling[component].hpa_name,
      hpa_prefix: $._config.autoscaling_hpa_prefix,
    };
    local desiredReplicas(targetType, multiplier) = |||
      sum by (scaledObject, metric) (
        # Using `max by ()` so that series churn doesn't break the promQL join
        max by (%(group_by)s, scaledObject, metric) (
          label_replace(
            keda_scaler_metrics_value{%(cluster_label)s=~"$cluster", exported_namespace=~"$namespace"},
            "namespace", "$1", "exported_namespace", "(.*)"
          )
        )
        / on (%(group_by)s, scaledObject, metric) group_left
        max by (%(group_by)s, scaledObject, metric) (
          label_replace(
            label_replace(
              kube_horizontalpodautoscaler_spec_target_metric{%(ns)s, horizontalpodautoscaler=~"%(hpa_name)s", metric_target_type="%(target_type)s"},
              "metric", "$1", "metric_name", "(.+)"
            ),
            "scaledObject", "$1", "horizontalpodautoscaler", "%(hpa_prefix)s(.*)"
          )
        )%(multiplier)s
      )
    ||| % (params { target_type: targetType, multiplier: multiplier });
    local timesCurrentReplicas = |||
      * on (%(group_by)s, scaledObject) group_left
      max by (%(group_by)s, scaledObject) (
        label_replace(
          kube_horizontalpodautoscaler_status_current_replicas{%(ns)s, horizontalpodautoscaler=~"%(hpa_name)s"},
          "scaledObject", "$1", "horizontalpodautoscaler", "%(hpa_prefix)s(.*)"
        )
      )
    ||| % params;

    $.panel('Desired replicas') +
    { fieldConfig+: { defaults+: { unit: 'short' } } } +
    $.queryPanel(
      '(\n%s)\nor\n(\n%s)' % [
        desiredReplicas('average', ''),
        desiredReplicas('value', std.strReplace('\n' + std.rstripChars(timesCurrentReplicas, '\n'), '\n', '\n  ')),
      ],
      '{{scaledObject}} {{metric}}',
    ) +
    $.panelDescription(
      'Desired replicas',
      |||
        The number of replicas each autoscaling trigger asks for: the scaling metric exposed by KEDA
        divided by its target. It ignores the min/max replicas constraints applied later by the HPA.
      |||
    ),

  // autoscalingFailuresPanel shows the rate at which KEDA fails to read the
  // scaling metrics, during which the autoscaler can't work properly. KEDA
  // 2.16 renamed keda_scaler_errors to keda_scaler_detail_errors_total, so
  // match both.
  autoscalingFailuresPanel(component)::
    $.panel('Autoscaler failures rate') +
    { fieldConfig+: { defaults+: { unit: 'short' } } } +
    $.queryPanel(
      |||
        sum by (scaledObject, metric) (
          # Using `max by ()` so duplicate series for the same trigger (both metric
          # names, or several KEDA operator replicas) aren't counted twice.
          max by (%(group_by)s, scaledObject, metric) (
            label_replace(
              rate({__name__=~"keda_scaler_detail_errors_total|keda_scaler_errors", %(cluster_label)s=~"$cluster", exported_namespace=~"$namespace"}[$__rate_interval]),
              "namespace", "$1", "exported_namespace", "(.+)"
            )
          )
          + on (%(group_by)s, scaledObject, metric) group_left
          # Using `max by ()` so that series churn doesn't break the promQL join
          max by (%(group_by)s, scaledObject, metric) (
            label_replace(
              label_replace(
                kube_horizontalpodautoscaler_spec_target_metric{%(ns)s, horizontalpodautoscaler=~"%(hpa_name)s"} * 0,
                "metric", "$1", "metric_name", "(.+)"
              ),
              "scaledObject", "$1", "horizontalpodautoscaler", "%(hpa_prefix)s(.*)"
            )
          )
        )
      ||| % {
        ns: $.namespaceMatcher(),
        cluster_label: $._config.per_cluster_label,
        group_by: $._config.group_by_cluster,
        hpa_name: $._config.autoscaling[component].hpa_name,
        hpa_prefix: $._config.autoscaling_hpa_prefix,
      },
      '{{scaledObject}} {{metric}} failures',
    ) +
    $.panelDescription(
      'Autoscaler failures rate',
      |||
        The rate of failures in the KEDA custom metrics API server. Whenever an error occurs, the KEDA custom
        metrics server is unable to query the scaling metric so the autoscaler wouldn't work properly.
      |||
    ),

  // autoscalingRow builds the "<title> - autoscaling" row for one of the
  // components defined in $._config.autoscaling.
  autoscalingRow(title, component)::
    local row = grafana.row('%s - autoscaling' % title)
                .addPanel($.autoscalingReplicasPanel(component));

    if $._config.autoscaling[component].keda_triggers then
      row
      .addPanel($.autoscalingDesiredReplicasPanel(component))
      .addPanel($.autoscalingFailuresPanel(component))
    else
      row,

  namespaceMatcher()::
    $._config.per_cluster_label + '=~"$cluster", namespace=~"$namespace"',

  containerCPUUsagePanel(title, containerName)::
    $.timeseriesPanel(title) +
    $.queryPanel([
      'sum by(pod) (rate(container_cpu_usage_seconds_total{%s,container=~"%s"}[$__rate_interval]))' % [$.namespaceMatcher(), containerName],
      'min(container_spec_cpu_quota{%s,container=~"%s"} / container_spec_cpu_period{%s,container=~"%s"})' % [$.namespaceMatcher(), containerName, $.namespaceMatcher(), containerName],
      'min(kube_pod_container_resource_requests{%s,container=~"%s", resource="cpu"} > 0)' % [$.namespaceMatcher(), containerName],
    ], ['{{pod}}', 'limit', 'request']) +
    {
      seriesOverrides: [
        {
          alias: 'limit',
          color: '#E02F44',
          fill: 0,
        },
        {
          alias: 'request',
          color: '#FCE300',
          fill: 0,
        },
      ],
      fieldConfig+: { defaults+: { unit: 'cores' } },
    },

  containerMemoryWorkingSetPanel(title, containerName)::
    $.timeseriesPanel(title) +
    $.queryPanel([
      'sum by(pod) (container_memory_working_set_bytes{%s,container=~"%s"})' % [$.namespaceMatcher(), containerName],
      'min(container_spec_memory_limit_bytes{%s,container=~"%s"} > 0)' % [$.namespaceMatcher(), containerName],
      'min(kube_pod_container_resource_requests{%s,container=~"%s", resource="memory"} > 0)' % [$.namespaceMatcher(), containerName],
    ], ['{{pod}}', 'limit', 'request']) +
    {
      seriesOverrides: [
        {
          alias: 'limit',
          color: '#E02F44',
          fill: 0,
        },
        {
          alias: 'request',
          color: '#FCE300',
          fill: 0,
        },
      ],
      fieldConfig+: { defaults+: { unit: 'bytes' } },
      yaxes: $.yaxes('bytes'),
    },

  goHeapInUsePanel(title, job)::
    $.timeseriesPanel(title) +
    $.queryPanel('sum by(instance) (go_memstats_heap_inuse_bytes{%s})' % job, '{{instance}}') +
    {
      fieldConfig+: { defaults+: { unit: 'bytes' } },
      yaxes: $.yaxes('bytes'),
    },

  newStatPanel(queries, legends='', unit='percentunit', decimals=1, thresholds=[], instant=false, novalue='')::
    super.queryPanel(queries, legends) + {
      type: 'stat',
      targets: [
        target {
          instant: instant,
          interval: '',

          // Reset defaults from queryPanel().
          format: null,
          intervalFactor: null,
          step: null,
        }
        for target in super.targets
      ],
      fieldConfig: {
        defaults: {
          color: { mode: 'thresholds' },
          decimals: decimals,
          thresholds: {
            mode: 'absolute',
            steps: thresholds,
          },
          noValue: novalue,
          unit: unit,
        },
        overrides: [],
      },
    },

  barGauge(queries, legends='', thresholds=[], unit='short', min=null, max=null)::
    super.queryPanel(queries, legends) + {
      type: 'bargauge',
      targets: [
        target {
          // Reset defaults from queryPanel().
          format: null,
          intervalFactor: null,
          step: null,
        }
        for target in super.targets
      ],
      fieldConfig: {
        defaults: {
          color: { mode: 'thresholds' },
          mappings: [],
          max: max,
          min: min,
          thresholds: {
            mode: 'absolute',
            steps: thresholds,
          },
          unit: unit,
        },
      },
      options: {
        displayMode: 'basic',
        orientation: 'horizontal',
        reduceOptions: {
          calcs: ['lastNotNull'],
          fields: '',
          values: false,
        },
      },
    },

  stack:: super.stack + {
    fieldConfig+: {
      defaults+: {
        custom+: {
          stacking: {
            mode: 'normal',
          },
        },
      },
    },
  },

}
