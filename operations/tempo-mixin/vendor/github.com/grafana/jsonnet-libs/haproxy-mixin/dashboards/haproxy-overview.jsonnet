local d = import 'dashboards.libsonnet';
local g = import 'github.com/grafana/dashboard-spec/_gen/7.0/jsonnet/grafana.libsonnet';


g.dashboard.new() + {
  title: 'HAProxy / Overview',

  panels:
    local headline = d.util.section(
      g.panel.row.new(title='Headline'),
      [
        d.panels.processUptime,
        d.panels.processCurrentConnections,
        d.panels.processMemoryAllocated,
        d.panels.processMemoryUsed,
      ]
    );
    local frontend = d.util.section(
      g.panel.row.new(title='Frontend'),
      [d.panels.frontendStatus],
      prevSection=headline,
      panelSize={ h: 8, w: 24 }
    );
    local backend = d.util.section(
      g.panel.row.new(title='Backend'),
      [d.panels.backendStatus],
      prevSection=frontend,
      panelSize={ h: 8, w: 24 }
    );
    local configuration = d.util.section(
      g.panel.row.new(title='Configuration'),
      [
        d.panels.processCount,
        d.panels.processThreads,
        d.panels.processConnectionLimit,
        d.panels.processFdLimit,
        d.panels.processSocketLimit,
        d.panels.processMemoryLimit,
        d.panels.processPipeLimit,
        d.panels.processConnectionRateLimit,
        d.panels.processSessionRateLimit,
        d.panels.processSslRateLimit,
      ],
      prevSection=backend
    );
    headline + frontend + backend + configuration,
  templating: {
    list: [
      d.templates.datasource,
      d.templates.instance,
      d.templates.job,
    ],
  },
  time: {
    from: 'now-6h',
    to: 'now',
  },
  timepicker: {
    hidden: false,
    refresh_intervals: [
      '5s',
      '10s',
      '30s',
      '1m',
      '5m',
      '15m',
      '30m',
      '1h',
      '2h',
      '1d',
    ],
  },
  uid: 'HAProxyOverview',
}
