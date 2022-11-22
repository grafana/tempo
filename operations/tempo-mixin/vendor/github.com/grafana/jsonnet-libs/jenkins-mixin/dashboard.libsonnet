local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');

local base_matcher = 'job=~"$job", instance=~"$instance"';

local queries = {
  http_200: 'rate(http_responseCodes_ok_total{' + base_matcher + '}[$__rate_interval])',
  http_201: 'rate(http_responseCodes_created_total{' + base_matcher + '}[$__rate_interval])',
  http_204: 'rate(http_responseCodes_noContent_total{' + base_matcher + '}[$__rate_interval])',
  http_304: 'rate(http_responseCodes_notModified_total{' + base_matcher + '}[$__rate_interval])',
  http_400: 'rate(http_responseCodes_badRequest_total{' + base_matcher + '}[$__rate_interval])',
  http_403: 'rate(http_responseCodes_forbidden_total{' + base_matcher + '}[$__rate_interval])',
  http_404: 'rate(http_responseCodes_notFound_total{' + base_matcher + '}[$__rate_interval])',
  http_500: 'rate(http_responseCodes_serverError_total{' + base_matcher + '}[$__rate_interval])',
  http_503: 'rate(http_responseCodes_serviceUnavailable_total{' + base_matcher + '}[$__rate_interval])',
  http_other: 'rate(http_responseCodes_other_total{' + base_matcher + '}[$__rate_interval])',
  http_durations_by_quantile: 'http_requests{' + base_matcher + '}',
  executor_count: 'jenkins_executor_count_value{' + base_matcher + '}',
  executor_free: 'jenkins_executor_free_value{' + base_matcher + '}',
  executor_inuse: 'jenkins_executor_in_use_value{' + base_matcher + '}',
  job_success_rate: 'rate(jenkins_runs_success_total{' + base_matcher + '}[$__rate_interval])',
  job_failure_rate: 'rate(jenkins_runs_failure_total{' + base_matcher + '}[$__rate_interval])',
  top5_longest_jobs: 'topk(5, max by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_last_build_duration_milliseconds", ' + base_matcher + '}))',
  jobs_blocked: 'jenkins_queue_blocked_value{' + base_matcher + '}',
  jobs_buildable: 'jenkins_queue_buildable_value{' + base_matcher + '}',
  jobs_pending: 'jenkins_queue_pending_value{' + base_matcher + '}',
  jobs_stuck: 'jenkins_queue_stuck_value{' + base_matcher + '}',
  build_total_by_job: 'sum by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_failed_build_count|$prefix\\\\_jenkins_builds_success_build_count", ' + base_matcher + '})',
  bottom5_healthy_job_build_total: 'sum by (jenkins_job) ({__name__=~"$prefix\\\\_jenkins_builds_failed_build_count|$prefix\\\\_?jenkins_builds_success_build_count", ' + base_matcher + '}) and on (jenkins_job) bottomk(5, avg by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_health_score", ' + base_matcher + '}))',
  bottom5_healthy_job_build_success_total: 'sum by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_success_build_count", ' + base_matcher + '}) and on (jenkins_job) bottomk(5, avg by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_health_score", ' + base_matcher + '}))',
  bottom5_healthy_job_build_failed_total: 'sum by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_failed_build_count", ' + base_matcher + '}) and on (jenkins_job) bottomk(5, avg by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_health_score", ' + base_matcher + '}))',
  bottom5_healthy_job_health: 'bottomk(5, avg by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_health_score", ' + base_matcher + '}))',
  build_duration_by_job: 'max by (jenkins_job) ({__name__=~"$prefix\\\\_?jenkins_builds_last_build_duration_milliseconds", ' + base_matcher + '})',
  nodes_online: 'jenkins_node_online_value{' + base_matcher + '}/jenkins_node_count_value{' + base_matcher + '}',
  plugins_active: 'jenkins_plugins_active{' + base_matcher + '}',
  plugins_inactive: 'jenkins_plugins_inactive{' + base_matcher + '}',
  plugins_failed: 'jenkins_plugins_failed{' + base_matcher + '}',
  plugins_withUpdate: 'jenkins_plugins_withUpdate{' + base_matcher + '}',
};

local inverse_colors = ['red', 'yellow', 'green'];

local stackstyle = {
  line: 1,
  fill: 5,
  fillGradient: 10,
};

// Templates
local ds_template = {
  current: {
    text: 'default',
    value: 'default',
  },
  hide: 0,
  label: 'Data Source',
  name: 'datasource',
  options: [],
  query: 'prometheus',
  refresh: 1,
  regex: '',
  type: 'datasource',
};

local job_template = grafana.template.new(
  'job',
  '$datasource',
  'label_values(up, job)',
  label='job',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local instance_template = grafana.template.new(
  'instance',
  '$datasource',
  'label_values(up{job=~"$job"}, instance)',
  label='instance',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local prefix_template = {
  current: {
    selected: false,
    text: '',
    value: 'default',
  },
  description: null,
  'error': null,
  hide: 0,
  label: null,
  name: 'prefix',
  options: [
    {
      selected: true,
      text: '',
      value: 'default',
    },
  ],
  query: 'default',
  skipUrlSync: false,
  type: 'textbox',
};

// Panels
local ui_reqs_panel = grafana.graphPanel.new(
                        'Web Request Rate by Status',
                        span=6,
                        datasource='$datasource',
                      ) +
                      g.queryPanel(
                        [queries.http_200, queries.http_201, queries.http_204, queries.http_304, queries.http_400, queries.http_403, queries.http_404, queries.http_500, queries.http_503, queries.http_other],
                        ['200 {{instance}}', '201 {{instance}}', '204 {{instance}}', '304 {{instance}}', '400 {{instance}}', '403 {{instance}}', '404 {{instance}}', '500 {{instance}}', '503 {{instance}}', 'other {{instance}}'],
                      );

local http_duration_by_quantile_panel = grafana.barGaugePanel.new(
                                          'HTTP Request Duration by Quantile',
                                          unit='s',
                                          datasource='$datasource',
                                          thresholds=[
                                            { color: 'blue', value: 0 },
                                          ],
                                        )
                                        .addTarget(
                                          grafana.prometheus.target(
                                            queries.http_durations_by_quantile,
                                            legendFormat='{{quantile}}'
                                          )
                                        ) +
                                        {
                                          options+: {
                                            reduceOptions: {
                                              values: false,
                                              calcs: ['mean'],
                                              fields: '',
                                            },
                                          },
                                          span: 6,
                                        };

local executor_count_panel = grafana.singlestat.new(
  'Total Executors',
  span=2,
  datasource='$datasource',
)
                             .addTarget(
  grafana.prometheus.target(queries.executor_count)
);

local executor_by_state_panel = grafana.pieChartPanel.new(
                                  'Executors By State',
                                  span=2,
                                  datasource='$datasource',
                                  showLegendPercentage=false,
                                  legendType='Under graph',
                                )
                                .addTarget(
                                  grafana.prometheus.target(queries.executor_free, legendFormat='free', instant=true)
                                )
                                .addTarget(
                                  grafana.prometheus.target(queries.executor_inuse, legendFormat='in-use', instant=true)
                                ) +
                                {
                                  type: 'piechart',
                                };

local job_success_rate_panel = grafana.statPanel.new(
                                 'Job Success Rate',
                                 datasource='$datasource',
                                 colorMode='background',
                               )
                               .addThreshold({ color: 'purple', value: 0 })
                               .addTarget(
                                 grafana.prometheus.target(queries.job_success_rate)
                               ) +
                               {
                                 span: 2,
                               };

local job_failure_rate_panel = grafana.statPanel.new(
                                 'Job Falure Rate',
                                 datasource='$datasource',
                                 colorMode='background',
                               )
                               .addThreshold({ color: 'red', value: 0 })
                               .addTarget(
                                 grafana.prometheus.target(queries.job_failure_rate)
                               ) +
                               {
                                 span: 2,
                               };

local nodes_online_panel = grafana.singlestat.new(
  'Build Nodes Online',
  datasource='$datasource',
  span=2,
  gaugeShow=true,
  format='percentunit',
  colors=inverse_colors,
  gaugeMaxValue=1,
  thresholds='.80,.90',
)
                           .addTarget(
  grafana.prometheus.target(queries.nodes_online, instant=true)
);

local plugins_by_state_panel = grafana.barGaugePanel.new(
                                 'Plugins by State',
                                 datasource='$datasource',
                                 thresholds=[{ color: 'purple', value: 0 }],
                               )
                               .addTargets([
                                 grafana.prometheus.target(queries.plugins_active, legendFormat='active', instant=true),
                                 grafana.prometheus.target(queries.plugins_inactive, legendFormat='inactive', instant=true),
                                 grafana.prometheus.target(queries.plugins_failed, legendFormat='failed', instant=true),
                                 grafana.prometheus.target(queries.plugins_withUpdate, legendFormat='with update', instant=true),
                               ]) +
                               {
                                 options+: {
                                   orientation: 'horizontal',
                                 },
                                 span: 2,
                               };

local job_queue_by_state_panel = grafana.graphPanel.new(
                                   'Job Queue by State',
                                   span=2,
                                   datasource='$datasource',
                                   min=0,
                                 ) +
                                 g.queryPanel(
                                   [queries.jobs_stuck, queries.jobs_blocked, queries.jobs_pending, queries.jobs_buildable],
                                   ['Stuck', 'Blocked', 'Pending', 'Buildable'],
                                 ) +
                                 g.stack + stackstyle;

local build_duration_by_job_panel = grafana.graphPanel.new(
                                      'Latest Build Duration by Job',
                                      span=4,
                                      datasource='$datasource',
                                    ) +
                                    g.queryPanel(
                                      [queries.build_duration_by_job],
                                      ['{{jenkins_job}}'],
                                    ) +
                                    {
                                      yaxes: g.yaxes('ms'),
                                    };

local top5_longest_jobs_panel = g.tablePanel(
  [queries.top5_longest_jobs],
  {
    jenkins_job: { alias: 'Job Name' },
    Value: { alias: 'Duration', unit: 'ms' },
  },
) + { span: 2, datasource: '$datasource', title: 'Top 5 Slowest Jobs' };

local bottom5_healthiest_jobs_panel = g.tablePanel(
  [queries.bottom5_healthy_job_health, queries.bottom5_healthy_job_build_failed_total, queries.bottom5_healthy_job_build_success_total, queries.bottom5_healthy_job_build_total],
  {
    jenkins_job: { alias: 'Job' },
    'Value #A': {
      alias: 'Health',
      unit: 'percent',
      // These don't actually do anything, because they aren't parsed/grok'd by the grafana-builder lib.
      // In a world where I had more time, I would update grafana-builder and/or grafonnet to use the latest
      // table visualizations and do this properly, but there are time constrains.
      thresholds: ['80', '90'],
      colorMode: 'cell',
      colors: ['red', 'orange', 'green'],
    },
    'Value #B': { alias: 'Failed' },
    'Value #C': { alias: 'Successful' },
    'Value #D': { alias: 'Total' },
  },
) + { span: 4, datasource: '$datasource', title: 'Least Healthy Jobs' };

// Manifested stuff starts here
{
  grafanaDashboards+:: {
    'jenkins.json':
      grafana.dashboard.new('Jenkins', uid='2UJijx65J')
      .addTemplates([
        ds_template,
        job_template,
        instance_template,
        prefix_template,
      ])

      // Overview Row
      .addRow(
        grafana.row.new('Overview')

        .addPanel(executor_count_panel)

        .addPanel(executor_by_state_panel)

        .addPanel(job_success_rate_panel)

        .addPanel(job_failure_rate_panel)

        .addPanel(nodes_online_panel)

        .addPanel(plugins_by_state_panel)
      )

      // Jobs
      .addRow(
        grafana.row.new('Jobs')

        .addPanel(bottom5_healthiest_jobs_panel)

        .addPanel(build_duration_by_job_panel)

        .addPanel(top5_longest_jobs_panel)

        .addPanel(job_queue_by_state_panel)
      )

      // System Row
      .addRow(
        grafana.row.new('Web UI')

        .addPanel(ui_reqs_panel)

        .addPanel(http_duration_by_quantile_panel)
      ),

  },
}
