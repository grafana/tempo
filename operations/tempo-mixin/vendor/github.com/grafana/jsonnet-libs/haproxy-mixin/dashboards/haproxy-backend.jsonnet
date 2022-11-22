local d = import 'dashboards.libsonnet';
local g = import 'github.com/grafana/dashboard-spec/_gen/7.0/jsonnet/grafana.libsonnet';

g.dashboard.new() + {
  title: 'HAProxy / Backend',

  panels:
    local servers = d.util.section(
      g.panel.row.new(title='Servers'),
      [d.panels.serverStatus],
      panelSize={ h: 8, w: 24 },
    );
    local requests = d.util.section(
      g.panel.row.new(title='Requests'),
      [
        d.panels.backendHttpRequestRate,
        d.panels.backendConnectionRate,
        d.panels.backendBytes,
      ],
      prevSection=servers,
      panelSize={ h: 6, w: 8 },
    );
    local errors = d.util.section(
      g.panel.row.new(title='Errors'),
      [
        d.panels.backendResponseErrorRate,
        d.panels.backendConnectionErrorRate,
        d.panels.backendInternalErrorRate,
      ],
      prevSection=requests,
      panelSize={ h: 6, w: 8 },
    );
    local duration = d.util.section(
      g.panel.row.new(title='Duration'),
      [
        d.panels.backendAverageDuration,
        d.panels.backendMaxDuration,
      ],
      prevSection=errors,
      panelSize={ h: 6, w: 8 },
    );
    servers + requests + errors + duration,

  templating: {
    list: [
      d.templates.datasource,
      d.templates.instance,
      d.templates.job,
      d.templates.backend,
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
  uid: 'HAProxyBackend',
}
