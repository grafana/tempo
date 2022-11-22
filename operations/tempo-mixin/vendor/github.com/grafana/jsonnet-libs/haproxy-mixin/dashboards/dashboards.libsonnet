local g = import 'github.com/grafana/dashboard-spec/_gen/7.0/jsonnet/grafana.libsonnet';

{
  _config+:: {
    // Grafana Prometheus data source name.
    datasource: 'prometheus',

    // Opinionated label matchers for Prometheus queries.
    instanceMatcher: 'instance=~"$instance"',
    jobMatcher: 'job=~"$job"',
    frontendMatcher: 'proxy=~"$frontend"',
    backendMatcher: 'proxy=~"$backend"',
    serverMatcher: 'server=~"$server"',
    baseMatchers: std.join(',', [self.instanceMatcher, self.jobMatcher]),
    frontendMatchers: std.join(',', [self.baseMatchers, self.frontendMatcher]),
    backendMatchers: std.join(',', [self.baseMatchers, self.backendMatcher]),
    serverMatchers: std.join(',', [self.backendMatchers, self.serverMatcher]),
  },

  util:: {
    addRefIds(targets):: std.mapWithIndex(function(i, e) e { refID: std.char(std.codepoint('A') + i) }, targets),

    // section is used to group multiple panels with common dimensions under a single uncollapsed row.
    section(row, panels, prevSection=null, panelSize={ h: 4, w: 6 })::
      local y = if prevSection != null then prevSection[std.length(prevSection) - 1].gridPos.y + 1 else 0;
      local id = if prevSection != null then prevSection[std.length(prevSection) - 1].id + 1 else 0;
      local dashboardWidth = 24;
      [
        row {
          collapsed: false,
          gridPos: { x: 0, y: y, h: 1, w: dashboardWidth },
          id: id,
        },
      ]
      + std.mapWithIndex(function(i, panel) panel {
        gridPos: {
          x: i * panelSize.w % dashboardWidth,
          y: y + (std.floor(i * panelSize.w / dashboardWidth) * panelSize.h),
          w: panelSize.w,
          h: panelSize.h,
        },
        id: id + i + 1,
      }, panels),
  },

  queries:: {
    processUptime: 'time() - haproxy_process_start_time_seconds{%s}' % $._config.baseMatchers,
    processCurrentConnections: 'haproxy_process_current_connections{%s}' % $._config.baseMatchers,
    processMemoryAllocated: 'haproxy_process_pool_allocated_bytes{%s}' % $._config.baseMatchers,
    processMemoryUsed: 'haproxy_process_pool_used_bytes{%s}' % $._config.baseMatchers,
    processMemoryFailures: 'haproxy_process_pool_failures_total{%s}' % $._config.baseMatchers,
    processThreads: 'haproxy_process_nbthread{%s}' % $._config.baseMatchers,
    processCount: 'haproxy_process_nbproc{%s}' % $._config.baseMatchers,
    processConnectionLimit: 'haproxy_process_max_connections{%s}' % $._config.baseMatchers,
    processFdLimit: 'haproxy_process_max_fds{%s}' % $._config.baseMatchers,
    processMemoryLimit: 'haproxy_process_max_memory_bytes{%s}' % $._config.baseMatchers,
    processSocketLimit: 'haproxy_process_max_sockets{%s}' % $._config.baseMatchers,
    processPipeLimit: 'haproxy_process_max_pipes{%s}' % $._config.baseMatchers,
    processConnectionRateLimit: 'haproxy_process_limit_connection_rate{%s}' % $._config.baseMatchers,
    processSessionRateLimit: 'haproxy_process_limit_session_rate{%s}' % $._config.baseMatchers,
    processSslRateLimit: 'haproxy_process_limit_ssl_rate{%s}' % $._config.baseMatchers,
    backendStatus: 'haproxy_backend_status{%s}' % $._config.baseMatchers,
    frontendStatus: 'haproxy_frontend_status{%s}' % $._config.baseMatchers,
    serverStatus: 'haproxy_server_status{%s}' % $._config.backendMatchers,
    backendBytesInRate: 'rate(haproxy_backend_bytes_in_total{%s}[$__rate_interval])' % $._config.backendMatchers,
    backendBytesOutRate: 'rate(haproxy_backend_bytes_out_total{%s}[$__rate_interval])' % $._config.backendMatchers,
    backendHttpRequestRate: 'rate(haproxy_backend_http_requests_total{%s}[$__rate_interval])' % $._config.backendMatchers,
    backendMaxQueueDuration: 'haproxy_backend_max_queue_time_seconds{%s}' % $._config.backendMatchers,
    backendMaxConnectDuration: 'haproxy_backend_max_connect_time_seconds{%s}' % $._config.backendMatchers,
    backendMaxResponseDuration: 'haproxy_backend_max_response_time_seconds{%s}' % $._config.backendMatchers,
    backendMaxTotalDuration: 'haproxy_backend_max_total_time_seconds{%s}' % $._config.backendMatchers,
    backendAverageQueueDuration: 'haproxy_backend_queue_time_average_seconds{%s}' % $._config.backendMatchers,
    backendAverageConnectDuration: 'haproxy_backend_connect_time_average_seconds{%s}' % $._config.backendMatchers,
    backendAverageResponseDuration: 'haproxy_backend_response_time_average_seconds{%s}' % $._config.backendMatchers,
    backendAverageTotalDuration: 'haproxy_backend_total_time_average_seconds{%s}' % $._config.backendMatchers,
    backendConnectionRate: 'rate(haproxy_backend_connection_attempts_total{%s}[$__rate_interval])' % $._config.backendMatchers,
    backendResponseErrorRate: 'rate(haproxy_backend_response_errors_total{%s}[$__rate_interval])' % $._config.backendMatchers,
    backendConnectionErrorRate: 'rate(haproxy_backend_connection_errors_total{%s}[$__rate_interval])' % $._config.backendMatchers,
    backendInternalErrorRate: 'rate(haproxy_backend_internal_errors_total{%s}[$__rate_interval])' % $._config.backendMatchers,
    frontendRequestErrorRate: 'rate(haproxy_frontend_request_errors_total{%s}[$__rate_interval])' % $._config.frontendMatchers,
    frontendHttpRequestRate: 'rate(haproxy_frontend_http_requests_total{%s}[$__rate_interval])' % $._config.frontendMatchers,
    frontendConnectionRate: 'rate(haproxy_frontend_connections_total{%s}[$__rate_interval])' % $._config.frontendMatchers,
    frontendInternalErrorRate: 'rate(haproxy_frontend_internal_errors_total{%s}[$__rate_interval])' % $._config.frontendMatchers,
    frontendBytesInRate: 'rate(haproxy_frontend_bytes_in_total{%s}[$__rate_interval])' % $._config.frontendMatchers,
    frontendBytesOutRate: 'rate(haproxy_frontend_bytes_out_total{%s}[$__rate_interval])' % $._config.frontendMatchers,
    frontendCacheSuccessRate: |||
      rate(haproxy_frontend_http_cache_lookups_total{%s}[5m])
      /
      rate(haproxy_frontend_http_cache_hits_total{%s}[5m])
    ||| % $._config.frontendMatchers,
    serverHttpResponseRate: 'rate(haproxy_server_http_responses_total{%s}[$__rate_interval])' % $._config.serverMatchers,
    serverMaxQueueDuration: 'haproxy_server_max_queue_time_seconds{%s}' % $._config.serverMatchers,
    serverMaxConnectDuration: 'haproxy_server_max_connect_time_seconds{%s}' % $._config.serverMatchers,
    serverMaxResponseDuration: 'haproxy_server_max_response_time_seconds{%s}' % $._config.serverMatchers,
    serverMaxTotalDuration: 'haproxy_server_max_total_time_seconds{%s}' % $._config.serverMatchers,
    serverAverageQueueDuration: 'haproxy_server_queue_time_average_seconds{%s}' % $._config.serverMatchers,
    serverAverageConnectDuration: 'haproxy_server_connect_time_average_seconds{%s}' % $._config.serverMatchers,
    serverAverageResponseDuration: 'haproxy_server_response_time_average_seconds{%s}' % $._config.serverMatchers,
    serverAverageTotalDuration: 'haproxy_server_total_time_average_seconds{%s}' % $._config.serverMatchers,
    serverConnectionRate: 'rate(haproxy_server_connection_attempts_total{%s}[$__rate_interval])' % $._config.serverMatchers,
    serverResponseErrorRate: 'rate(haproxy_server_response_errors_total{%s}[$__rate_interval])' % $._config.serverMatchers,
    serverConnectionErrorRate: 'rate(haproxy_server_connection_errors_total{%s}[$__rate_interval])' % $._config.serverMatchers,
    serverInternalErrorRate: 'rate(haproxy_server_internal_errors_total{%s}[$__rate_interval])' % $._config.serverMatchers,
    serverBytesInRate: 'rate(haproxy_server_bytes_in_total{%s}[$__rate_interval])' % $._config.serverMatchers,
    serverBytesOutRate: 'rate(haproxy_server_bytes_out_total{%s}[$__rate_interval])' % $._config.serverMatchers,
  },

  // panels compose queries into useful Grafana panels.
  panels:: {
    // infoMixin colors stat panel fields blue so as to represent information that has neutral connotations.
    infoMixin:: {
      fieldConfig+: {
        defaults+: {
          thresholds+: {
            mode+: 'absolute',
            steps: [{ color: 'blue', value: null }],
          },
        },
      },
    },

    // zeroUnsetMixin relabels the value '0' to 'unset' to indicate the configuration value has not been set.
    zeroUnsetMixin:: {
      fieldConfig+: {
        defaults+: {
          mappings: [{ id: 1, type: 1, text: 'unset', value: 0 }],
        },
      },
    },

    // statusUpDownMixin relabels up and down status values and colors them for easy recognition.
    statusUpDownMixin: {
      fieldConfig+: {
        overrides+: [{
          matcher+: { id: 'byName', options: 'Status' },
          properties: [
            {
              id: 'mappings',
              value: [
                { id: 1, type: 1, text: 'Down', value: '0' },
                { id: 2, type: 1, text: 'Up', value: '1' },
              ],
            },
            {
              id: 'custom.displayMode',
              value: 'color-background',
            },
            {
              id: 'thresholds',
              value: {
                mode: 'absolute',
                steps: [
                  { color: 'rgba(0,0,0,0)', value: null },
                  { color: 'red', value: 0 },
                  { color: 'green', value: 1 },
                ],
              },
            },
          ],
        }],
      },
    },

    processUptime:
      g.panel.stat.new()
      + $.panels.infoMixin {
        title: 'Uptime',
        datasource: '$datasource',
        description: 'Process uptime',
        fieldConfig+: { defaults+: { unit: 's' } },
        targets: $.util.addRefIds([{ expr: $.queries.processUptime }]),
      },

    processCurrentConnections:
      g.panel.stat.new()
      + $.panels.infoMixin {
        title: 'Current connections',
        datasource: '$datasource',
        description: 'Number of active sessions',
        targets: $.util.addRefIds([{ expr: $.queries.processCurrentConnections }]),
      },

    processMemoryAllocated:
      g.panel.stat.new()
      + $.panels.infoMixin {
        title: 'Memory allocated',
        datasource: '$datasource',
        description: 'Total amount of memory allocated in pools',
        fieldConfig+: { defaults+: { unit: 'bytes' } },
        targets: $.util.addRefIds([{ expr: $.queries.processMemoryAllocated }]),
      },

    processMemoryUsed:
      g.panel.stat.new()
      + $.panels.infoMixin {
        title: 'Memory used',
        datasource: '$datasource',
        description: 'Total amount of memory used in pools',
        fieldConfig+: { defaults+: { unit: 'bytes' } },
        targets: $.util.addRefIds([{ expr: $.queries.processMemoryUsed }]),
      },

    processThreads:
      g.panel.stat.new()
      + $.panels.infoMixin {
        title: 'Threads',
        datasource: '$datasource',
        description: 'Configured number of threads',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processThreads }]),
      },

    processCount:
      g.panel.stat.new()
      + $.panels.infoMixin {
        title: 'Processes',
        datasource: '$datasource',
        description: 'Configured number of processes',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processCount }]),
      },

    processConnectionLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'Connections limit',
        datasource: '$datasource',
        description: 'Configured maximum number of concurrent connections',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processConnectionLimit }]),
      },

    processMemoryLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'Memory limit',
        datasource: '$datasource',
        description: 'Per-process memory limit',
        fieldConfig+: { defaults+: { unit: 'bytes' } },
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processMemoryLimit }]),
      },

    processFdLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'File descriptors limit',
        datasource: '$datasource',
        description: 'Maximum number of open file descriptors',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processFdLimit }]),
      },

    processSocketLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'Socket limit',
        datasource: '$datasource',
        description: 'Maximum number of open sockets',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processSocketLimit }]),
      },

    processPipeLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'Pipe limit',
        datasource: '$datasource',
        description: 'Maximum number of pipes',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processPipeLimit }]),
      },

    processConnectionRateLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'Connection rate limit',
        datasource: '$datasource',
        description: 'Maximum number of connections per second',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processConnectionRateLimit }]),
      },

    processSessionRateLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'Session rate limit',
        datasource: '$datasource',
        description: 'Maximum number of sessions per second',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processSessionRateLimit }]),
      },

    processSslRateLimit:
      g.panel.stat.new()
      + $.panels.infoMixin
      + $.panels.zeroUnsetMixin {
        title: 'SSL session rate limit',
        datasource: '$datasource',
        description: 'Maximum number of SSL sessions per second',
        options+: { graphMode: 'none' },
        targets: $.util.addRefIds([{ expr: $.queries.processSslRateLimit }]),
      },

    frontendStatus:
      g.panel.table.new()
      + $.panels.statusUpDownMixin {
        datasource: '$datasource',
        fieldConfig+: {
          defaults+: {
            links: [{
              title: 'Frontend',
              datasource: '$datasource',
              url: '/d/HAProxyFrontend/haproxy-frontend?${__all_variables}&var-frontend=${__data.fields.Frontend}',
            }],
          },

        },
        options: { sortBy: [{ displayName: 'Status', desc: false }] },
        targets: $.util.addRefIds([{
          expr: $.queries.frontendStatus,
          format: 'table',
          instant: true,
        }]),
        transformations: [
          {
            id: 'organize',
            options: {
              excludeByName: {
                Time: true,
                __name__: true,
              },
              renameByName: {
                instance: 'Instance',
                job: 'Job',
                proxy: 'Frontend',
                Value: 'Status',
              },
            },
          },
        ],
      },

    backendStatus:
      g.panel.table.new()
      + $.panels.statusUpDownMixin {
        datasource: '$datasource',
        fieldConfig+: {
          defaults+: {
            links: [{
              title: 'Backend',
              datasource: '$datasource',
              url: '/d/HAProxyBackend/haproxy-backend?${__all_variables}&var-backend=${__data.fields.Backend}',
            }],
          },
        },
        targets: $.util.addRefIds([{
          expr: $.queries.backendStatus,
          format: 'table',
          instant: true,
        }]),
        transformations: [
          {
            id: 'organize',
            options: {
              excludeByName: {
                Time: true,
                __name__: true,
              },
              renameByName: {
                instance: 'Instance',
                job: 'Job',
                proxy: 'Backend',
                Value: 'Status',
              },
            },
          },
        ],
      },

    serverStatus::
      g.panel.table.new()
      + $.panels.statusUpDownMixin {
        datasource: '$datasource',
        fieldConfig+: {
          defaults+: {
            links: [{
              title: 'Server',
              datasource: '$datasource',
              url: '/d/HAProxyServer/haproxy-server?${__all_variables}&var-server=${__data.fields.Server}',
            }],
          },
        },
        options: { sortBy: [{ displayName: 'Status', desc: false }] },
        targets: $.util.addRefIds([{
          expr: $.queries.serverStatus,
          format: 'table',
          instant: true,
        }]),
        transformations: [
          {
            id: 'organize',
            options: {
              excludeByName: {
                Time: true,
                __name__: true,
              },
              renameByName: {
                instance: 'Instance',
                job: 'Job',
                proxy: 'Backend',
                server: 'Server',
                Value: 'Status',
              },
            },
          },
        ],
      },

    frontendCacheSuccessRate:
      g.panel.graph.new() + {
        title: 'Cache success',
        datasource: '$datasource',
        description: 'Percentage of HTTP cache hits.',
        fieldConfig: { defaults: { unit: 'reqps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.frontendCacheSuccessRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    frontendRequestErrorRate:
      g.panel.graph.new() + {
        title: 'Requests',
        datasource: '$datasource',
        description: 'Request errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.frontendRequestErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        stack: true,
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    frontendHttpRequestRate:
      g.panel.graph.new() + {
        title: 'HTTP',
        datasource: '$datasource',
        description: 'HTTP requests per second',
        fieldConfig: { defaults: { unit: 'reqps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.frontendHttpRequestRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    frontendConnectionRate:
      g.panel.graph.new() + {
        title: 'Connections',
        datasource: '$datasource',
        description: 'Connections per second',
        fieldConfig: { defaults: { unit: 'connps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.frontendConnectionRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    frontendInternalErrorRate:
      g.panel.graph.new() + {
        title: 'Internal',
        datasource: '$datasource',
        description: 'Internal errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.frontendInternalErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        stack: true,
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    frontendBytes:
      g.panel.graph.new() + {
        title: 'Bytes in/out',
        datasource: '$datasource',
        fieldConfig: { defaults: { unit: 'bytes' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.frontendBytesInRate,
            legendFormat: '{{proxy}}:in',
          },
          {
            expr: $.queries.frontendBytesOutRate,
            legendFormat: '{{proxy}}:out',
          },
        ]),
        seriesOverrides: [{ alias: '/.*out.*/', transform: 'negative-Y' }],
      },

    backendHttpRequestRate:
      g.panel.graph.new() + {
        title: 'HTTP',
        datasource: '$datasource',
        description: 'HTTP requests per second. There will be no data for backends using tcp mode.',
        fieldConfig: { defaults: { unit: 'reqps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendHttpRequestRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    backendMaxDuration:
      g.panel.graph.new() {
        title: 'Max duration',
        datasource: '$datasource',
        description: 'Max duration for last 1024 successful connections',
        fieldConfig: { defaults: { unit: 's' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendMaxQueueDuration,
            legendFormat: '{{proxy}}:max queue time',
          },
          {
            expr: $.queries.backendMaxConnectDuration,
            legendFormat: '{{proxy}}:max connect time',
          },
          {
            expr: $.queries.backendMaxResponseDuration,
            legendFormat: '{{proxy}}:max response time',
          },
          {
            expr: $.queries.backendMaxTotalDuration,
            legendFormat: '{{proxy}}:max total time',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    backendAverageDuration:
      g.panel.graph.new() {
        title: 'Average duration',
        datasource: '$datasource',
        description: 'Average duration for last 1024 successful connections',
        fieldConfig: { defaults: { unit: 's' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendAverageQueueDuration,
            legendFormat: '{{proxy}}:avg queue time',
          },
          {
            expr: $.queries.backendAverageConnectDuration,
            legendFormat: '{{proxy}}:avg connect time',
          },
          {
            expr: $.queries.backendAverageResponseDuration,
            legendFormat: '{{proxy}}:avg response time',
          },
          {
            expr: $.queries.backendAverageTotalDuration,
            legendFormat: '{{proxy}}:avg total time',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    backendConnectionRate:
      g.panel.graph.new() {
        title: 'Connection',
        datasource: '$datasource',
        description: 'Attempted connections per second',
        fieldConfig: { defaults: { unit: 'connps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendConnectionRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    backendResponseErrorRate:
      g.panel.graph.new() {
        title: 'HTTP',
        datasource: '$datasource',
        description: 'HTTP response errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendResponseErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    backendConnectionErrorRate:
      g.panel.graph.new() {
        title: 'Connection',
        datasource: '$datasource',
        description: 'Connection errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendConnectionErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        stack: true,
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    backendInternalErrorRate:
      g.panel.graph.new() {
        title: 'Internal',
        datasource: '$datasource',
        description: 'Internal errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendInternalErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        stack: true,
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    backendBytes:
      g.panel.graph.new() {
        title: 'Bytes in/out',
        datasource: '$datasource',
        fieldConfig: { defaults: { unit: 'bytes' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.backendBytesInRate,
            legendFormat: '{{proxy}}:in',
          },
          {
            expr: $.queries.backendBytesOutRate,
            legendFormat: '{{proxy}}:out',
          },
        ]),
        seriesOverrides: [{ alias: '/.*out.*/', transform: 'negative-Y' }],
      },

    serverHttpResponseRate:
      g.panel.graph.new() + {
        title: 'HTTP Response',
        datasource: '$datasource',
        description: 'HTTP responses per second. There will be no data for servers using tcp mode.',
        fieldConfig: { defaults: { unit: 'reqps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverHttpResponseRate,
            legendFormat: '{{proxy}}:{{code}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    serverMaxDuration:
      g.panel.graph.new() + {
        title: 'Max duration',
        datasource: '$datasource',
        description: 'Max duration for last 1024 succesful connections',
        fieldConfig: { defaults: { unit: 's' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverMaxQueueDuration,
            legendFormat: '{{proxy}}:max queue time',
          },
          {
            expr: $.queries.serverMaxConnectDuration,
            legendFormat: '{{proxy}}:max connect time',
          },
          {
            expr: $.queries.serverMaxResponseDuration,
            legendFormat: '{{proxy}}:max response time',
          },
          {
            expr: $.queries.serverMaxTotalDuration,
            legendFormat: '{{proxy}}:max total time',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    serverAverageDuration:
      g.panel.graph.new() + {
        title: 'Average duration',
        datasource: '$datasource',
        description: 'Average duration for last 1024 succesful connections',
        fieldConfig: { defaults: { unit: 's' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverAverageQueueDuration,
            legendFormat: '{{proxy}}:avg queue time',
          },
          {
            expr: $.queries.serverAverageConnectDuration,
            legendFormat: '{{proxy}}:avg connect time',
          },
          {
            expr: $.queries.serverAverageResponseDuration,
            legendFormat: '{{proxy}}:avg response time',
          },
          {
            expr: $.queries.serverAverageTotalDuration,
            legendFormat: '{{proxy}}:avg total time',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    serverConnectionRate:
      g.panel.graph.new() + {
        title: 'Connection',
        datasource: '$datasource',
        description: 'Attempted connections per second',
        fieldConfig: { defaults: { unit: 'connps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverConnectionRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    serverResponseErrorRate:
      g.panel.graph.new() + {
        title: 'HTTP Response',
        datasource: '$datasource',
        description: 'Response errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverResponseErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    serverConnectionErrorRate:
      g.panel.graph.new() + {
        title: 'Connection',
        datasource: '$datasource',
        description: 'Connection errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverConnectionErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        stack: true,
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    serverInternalErrorRate:
      g.panel.graph.new() + {
        title: 'Internal',
        datasource: '$datasource',
        description: 'Internal errors per second',
        fieldConfig: { defaults: { unit: 'errps' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverInternalErrorRate,
            legendFormat: '{{proxy}}',
          },
        ]),
        stack: true,
        yaxes: [{ min: 0 }, { min: 0 }],
      },

    serverBytes:
      g.panel.graph.new() + {
        title: 'Bytes in/out',
        datasource: '$datasource',
        fieldConfig: { defaults: { unit: 'bytes' } },
        targets: $.util.addRefIds([
          {
            expr: $.queries.serverBytesInRate,
            legendFormat: '{{proxy}}:{{server}}:in',
          },
          {
            expr: $.queries.serverBytesOutRate,
            legendFormat: '{{proxy}}:{{server}}:out',
          },
        ]),
        seriesOverrides: [{ alias: '/.*out.*/', transform: 'negative-Y' }],
      },
  },

  // templates can be used as Grafana template variables.
  templates:: {
    datasource: g.template.datasource.new() + {
      name: 'datasource',
      label: 'Data Source',
      query: $._config.datasource,
      current: {
        selected: false,
        text: 'prometheus',
        value: 'prometheus',
      },
    },
    instance: g.template.query.new() + {
      name: 'instance',
      datasource: '$datasource',
      definition: 'label_values(haproxy_process_start_time_seconds, instance)',
      includeAll: true,
      multi: true,
      query: 'label_values(haproxy_process_start_time_seconds, instance)',
      refresh: 1,
    },
    job: g.template.query.new() + {
      name: 'job',
      datasource: '$datasource',
      definition: 'label_values(haproxy_process_start_time_seconds, job)',
      includeAll: true,
      multi: true,
      query: 'label_values(haproxy_process_start_time_seconds, job)',
      refresh: 1,
    },
    backend: g.template.query.new() + {
      name: 'backend',
      datasource: '$datasource',
      definition: 'label_values(haproxy_backend_status, proxy)',
      includeAll: true,
      multi: true,
      query: 'label_values(haproxy_backend_status, proxy)',
      refresh: 1,
    },
    frontend: g.template.query.new() + {
      name: 'frontend',
      datasource: '$datasource',
      definition: 'label_values(haproxy_frontend_status, proxy)',
      includeAll: true,
      multi: true,
      query: 'label_values(haproxy_frontend_status, proxy)',
      refresh: 1,
    },
    server: g.template.query.new() + {
      name: 'server',
      datasource: '$datasource',
      definition: 'label_values(haproxy_server_status, server)',
      includeAll: true,
      multi: true,
      query: 'label_values(haproxy_server_status, server)',
      refresh: 1,
    },
  },
}
