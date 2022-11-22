local d = import 'dashboards.libsonnet';
local g = import 'github.com/grafana/dashboard-spec/_gen/7.0/jsonnet/grafana.libsonnet';

g.dashboard.new() + {
  title: 'HAProxy / Frontend',

  panels:
    local requests = d.util.section(
      g.panel.row.new(title='Requests'),
      [
        d.panels.frontendHttpRequestRate,
        d.panels.frontendConnectionRate,
        d.panels.frontendBytes,
      ],
      panelSize={ h: 6, w: 8 },
    );
    local errors = d.util.section(
      g.panel.row.new(title='Errors'),
      [
        d.panels.frontendRequestErrorRate,
        d.panels.frontendInternalErrorRate,
      ],
      prevSection=requests,
      panelSize={ h: 6, w: 8 },
    );
    requests + errors,

  templating: {
    list: [
      d.templates.datasource,
      d.templates.instance,
      d.templates.job,
      d.templates.frontend,
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
  uid: 'HAProxyFrontend',
}
