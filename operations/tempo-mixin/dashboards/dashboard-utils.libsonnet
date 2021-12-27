local grafana = import 'grafana-builder/grafana.libsonnet';
local utils = import 'mixin-utils/utils.libsonnet';

grafana {
  // Override the dashboard constructor to add:
  // - default tags,
  // - some links that propagate the selected cluster.
  dashboard(title)::
    super.dashboard(title) + {
      addClusterSelectorTemplates()::
        local d = self {
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

        d.addMultiTemplate('cluster', 'tempo_build_info', 'cluster')
        .addMultiTemplate('namespace', 'tempo_build_info', 'namespace', allValue=null),
    },

  jobMatcher(job)::
    'cluster=~"$cluster", job=~"($namespace)/%s"' % job,

  queryPanel(queries, legends, legendLink=null)::
    super.queryPanel(queries, legends, legendLink) + {
      targets: [
        target {
          interval: '1m',
        }
        for target in super.targets
      ],
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
    },

  // fork of grafana latency panel with additional_grouping added
  latencyPanel(metricName, selector, multiplier='1e3', additional_grouping=''):: {
    nullPointMode: 'null as zero',
    targets: [
      {
        expr: 'histogram_quantile(0.99, sum(rate(%s_bucket%s[$__interval])) by (le,%s)) * %s' % [metricName, selector, additional_grouping, multiplier],
        format: 'time_series',
        intervalFactor: 2,
        legendFormat: '{{route}} 99th',
        refId: 'A',
        step: 10,
        interval: '1m',
      },
      {
        expr: 'histogram_quantile(0.50, sum(rate(%s_bucket%s[$__interval])) by (le,%s)) * %s' % [metricName, selector, additional_grouping, multiplier],
        format: 'time_series',
        intervalFactor: 2,
        legendFormat: '{{route}} 50th',
        refId: 'B',
        step: 10,
        interval: '1m',
      },
      {
        expr: 'sum(rate(%s_sum%s[$__interval])) by (%s) * %s / sum(rate(%s_count%s[$__interval])) by (%s)' % [metricName, selector, additional_grouping, multiplier, metricName, selector, additional_grouping],
        format: 'time_series',
        intervalFactor: 2,
        legendFormat: '{{route}} Average',
        refId: 'C',
        step: 10,
        interval: '1m',
      },
    ],
    yaxes: $.yaxes('ms'),
  },

  namespaceMatcher()::
    'cluster=~"$cluster", namespace=~"$namespace"',

  containerCPUUsagePanel(title, containerName)::
    $.panel(title) +
    $.queryPanel([
      'sum by(pod) (rate(container_cpu_usage_seconds_total{%s,container="%s"}[$__interval]))' % [$.namespaceMatcher(), containerName],
      'min(container_spec_cpu_quota{%s,container="%s"} / container_spec_cpu_period{%s,container="%s"})' % [$.namespaceMatcher(), containerName, $.namespaceMatcher(), containerName],
    ], ['{{pod}}', 'limit']) +
    {
      seriesOverrides: [
        {
          alias: 'limit',
          color: '#E02F44',
          fill: 0,
        },
      ],
    },

  containerMemoryWorkingSetPanel(title, containerName)::
    $.panel(title) +
    $.queryPanel([
      'sum by(pod) (container_memory_working_set_bytes{%s,container="%s"})' % [$.namespaceMatcher(), containerName],
      'min(container_spec_memory_limit_bytes{%s,container="%s"} > 0)' % [$.namespaceMatcher(), containerName],
    ], ['{{pod}}', 'limit']) +
    {
      seriesOverrides: [
        {
          alias: 'limit',
          color: '#E02F44',
          fill: 0,
        },
      ],
      yaxes: $.yaxes('bytes'),
    },

  goHeapInUsePanel(title, job)::
    $.panel(title) +
    $.queryPanel('sum by(instance) (go_memstats_heap_inuse_bytes{%s})' % job, '{{instance}}') +
    { yaxes: $.yaxes('bytes') },

}
