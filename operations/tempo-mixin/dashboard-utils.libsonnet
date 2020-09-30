local utils = import 'mixin-utils/utils.libsonnet';
local grafana = import 'grafana-builder/grafana.libsonnet';

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
         .addMultiTemplate('namespace', 'tempo_build_info', 'namespace'),
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

  latencyPanel(metricName, selector, multiplier='1e3')::
    super.latencyPanel(metricName, selector, multiplier) + {
      targets: [
        target {
          interval: '1m',
        }
        for target in super.targets
      ],
    },
}