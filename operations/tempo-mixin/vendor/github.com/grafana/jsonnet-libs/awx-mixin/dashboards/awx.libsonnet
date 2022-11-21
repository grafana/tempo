local grafana = import 'grafonnet/grafana.libsonnet';
local dashboard = grafana.dashboard;
local row = grafana.row;
local prometheus = grafana.prometheus;
local graphPanel = grafana.graphPanel;
local statPanel = grafana.statPanel;
local pieChartPanel = grafana.pieChartPanel;
local it = import 'instancetable.libsonnet';
local ct = import 'clustertable.libsonnet';

local dashboardUid = 'eqcdR8HDA';

local matcher = 'job=~"$job", instance=~"$instance"';

local queries = {
  awx_system_info: 'awx_system_info{' + matcher + '}',
  orgs_total: 'awx_organizations_total{' + matcher + '}',
  users_total: 'awx_users_total{' + matcher + '}',
  teams_total: 'awx_teams_total{' + matcher + '}',
  inventories_total: 'awx_inventories_total{' + matcher + '}',
  projects_total: 'awx_projects_total{' + matcher + '}',
  job_templates_total: 'awx_job_templates_total{' + matcher + '}',
  workflow_job_templates_total: 'awx_workflow_job_templates_total{' + matcher + '}',
  active_hosts: 'awx_hosts_total{type="active", ' + matcher + '}',
  inactive_hosts: 'awx_hosts_total{type="total", ' + matcher + '} - ' + queries.active_hosts,
  schedules_total: 'awx_schedules_total{' + matcher + '}',
  sessions_by_type: 'sum by (type) (awx_sessions_total{' + matcher + '})',

  license_free: 'awx_license_instance_free{' + matcher + '}',
  license_total: 'awx_license_instance_total{' + matcher + '}',

  instance_info: 'awx_instance_info{' + matcher + '}',
  instance_cpu: 'sum by (hostname) (awx_instance_cpu{' + matcher + '})',
  instance_memory: 'sum by (hostname) (awx_instance_memory{' + matcher + '})',
  instance_capacity: 'sum by (hostname) (awx_instance_capacity{' + matcher + '})',
  instance_consumed_capacity: 'sum by (hostname) (awx_instance_consumed_capacity{' + matcher + '})',
  instance_consumed_capacity_percent: queries.instance_consumed_capacity + ' / ' + queries.instance_capacity,

  launch_type_total: 'sum by (launch_type) (awx_instance_launch_type_total{' + matcher + '})',
  running_job_count: 'awx_running_jobs_total{' + matcher + '}',
  pending_job_count: 'awx_running_jobs_total{' + matcher + '}',
  job_status_rate: 'irate(awx_status_total{' + matcher + '}[$__rate_interval])',
  job_launch_type_total: 'sum by (launch_type) (awx_instance_launch_type_total{' + matcher + '})',
  job_status_total: 'sum by (status) (awx_instance_status_total{' + matcher + '})',
  job_launch_type_rate: 'sum by (launch_type) (irate(awx_instance_launch_type_total{' + matcher + '}[$__rate_interval]))',
  instance_job_status_rate: 'sum by (status, node) (irate(awx_instance_status_total{' + matcher + '}[$__rate_interval]))',
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

local job_template =
  grafana.template.new(
    'job',
    '$datasource',
    'label_values(awx_system_info, job)',
    label='job',
    refresh='load',
    multi=true,
    includeAll=true,
    allValues='.+',
    sort=1,
  );

local instance_template =
  grafana.template.new(
    'instance',
    '$datasource',
    'label_values(awx_system_info{job=~"$job"}, instance)',
    label='instance',
    refresh='load',
    multi=true,
    includeAll=true,
    allValues='.+',
    sort=1,
  );

// Local styles and overrides
local piechartupdate =
  {
    type: 'piechart',
    options: {
      reduceOptions: {
        values: false,
        calcs: [
          'lastNotNull',
        ],
        fields: '',
      },
      pieType: 'donut',
      tooltip: {
        mode: 'single',
      },
      legend: {
        displayMode: 'table',
        placement: 'right',
        values: [
          'value',
          'percent',
        ],
      },
    },
  };

// Utilities
/* This is a helper function to allow a panel to reference a query from another panel, reducing the cost of loading a dashboard
by not repeating queries.

The `panels` input parameter is expected to be a flat array of panels. I.E. No panels which themselves have a "panels" property, as used by the old rows pattern.
This does require that you use rigid `gridPos` rather than the default behavior of the `row` and `dashboard` libs of grafonnet.

Panels may either announce themselves as a publisher of shared queries, or specify that they subscribe to one or more queries.

Publishing panels must have a hidden property named `target_publisher` which can be set to any value, this function only checks
that it is present.

Subscribing panels must have a hidden property named `target_subscriber` which is an array of queries to match from publishing panels.

Example:
```
local query = 'up{job=~"$job"}';
local query1 = 'foo{job=~"$job"}';

local pub_panel = statPanel.new('zero').addTarget(grafana.prometheus.target(query)) + { target_publisher:: true };
local pub_panel1 = statPanel.new('one').addTarget(grafana.prometheus.target(query)) + { target_publisher:: true };

local sub_panel = statPanel.new('sub') + { target_subscriber: [query, query1] };

local rendered_dash =
  dashboard.new('dash')
  .addPanels([
    row.new('Dont nest panels in rows'),
    pub_panel,
    pub_panel1,
    sub_panel,
  ]);

{
  grafanaDashboards+:: {
    'dashboard.json':
      rendered_dash
      + target_matchmake(rendered_dash.panels),
  },
}
```
*/
local target_matchmake(panels) = {
  pubMap:: {
    [pubT.expr]: { panel: pubP, target: pubT }
    for pubP in std.filter(function(p) std.objectHasAll(p, 'target_publisher'), panels)
    for pubT in pubP.targets
  },
  panels:
    std.map(
      function(p)
        if std.objectHasAll(p, 'target_subscriber') then
          p {
            datasource: '-- Dashboard --',
            // TODO: In a downstream project, the grafana SDK is used for some evaluation, and extracting data.
            // As a result of a change to the grafana schema (covered in this PR -> https://github.com/grafana-tools/sdk/pull/190)
            // the only acceptable form of datasource is a string. Once that is sorted out, this can use the proper form which is
            // commented out below
            // datasource: {
            //   uid: '-- Dashboard --',
            //   type: 'datasource',
            // },
            targets+: std.map(function(sub) {
              panelId: $.pubMap[sub].panel.id,
              refId: $.pubMap[sub].target.refId,
              // This is probably redundant since we're overriding the datasource for the panel
              // datasource: {
              //   uid: '-- Dashboard --',
              //   type: 'datasource',
              // },
            }, p.target_subscriber),
          }
        else p, panels
    ),
};

// Panels
local cluster_objs =
  statPanel.new(
    'Cluster Objects',
    datasource='$datasource',
    colorMode='background',
    graphMode='none',
    justifyMode='center',
    orientation='vertical',
  )
  .addTargets([
    grafana.prometheus.target(queries.orgs_total, legendFormat='Orgs'),
    grafana.prometheus.target(queries.teams_total, legendFormat='Teams'),
    grafana.prometheus.target(queries.users_total, legendFormat='Users'),
    grafana.prometheus.target(queries.inventories_total, legendFormat='Inventories'),
    grafana.prometheus.target(queries.projects_total, legendFormat='Projects'),
    grafana.prometheus.target(queries.schedules_total, legendFormat='Schedules'),
  ]) +
  {
    fieldConfig+: {
      defaults+: {
        color: { mode: 'continuous-blues' },
      },
    },
  };

// This should be a timeseries since it will likely change over time
local sessions_pie =
  graphPanel.new('Sessions', datasource='$datasource')
  .addTarget(grafana.prometheus.target(queries.sessions_by_type, legendFormat='{{type}}'))
  + {
    fieldConfig+: {
      overrides+: [
        {
          matcher: {
            id: 'byName',
            options: 'anonymous',
          },
          properties: [
            {
              id: 'displayName',
              value: 'Anonymous',
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'user',
          },
          properties: [
            {
              id: 'displayName',
              value: 'Authenticated',
            },
          ],
        },
      ],
    },
    transformations: [
      {
        id: 'filterFieldsByName',
        options: {
          include: {
            names: [
              'Time',
              'anonymous',
              'user',
            ],
          },
        },
      },
    ],
  };

local baseurl_stat =
  statPanel.new(
    'Tower Base URL',
    colorMode='background',
    graphMode='none',
    justifyMode='center',
  )
  {
    target_subscriber:: [queries.awx_system_info],
    options+: {
      reduceOptions: {
        values: false,
        calcs: [
          'lastNotNull',
        ],
        fields: '/^tower_url_base$/',
      },
    },
    fieldConfig+: {
      defaults+: {
        color: { mode: 'fixed', fixedColor: 'purple' },
        links: [
          {
            title: '',
            url: '${__value.text}',
            targetBlank: true,
          },
        ],
      },
    },
  };

local hosts_pie =
  pieChartPanel.new('Hosts', datasource='$datasource')
  .addTarget(grafana.prometheus.target(queries.active_hosts, legendFormat='Active'))
  .addTarget(grafana.prometheus.target(queries.inactive_hosts, legendFormat='Inactive'))
  + piechartupdate;

local tower_ver_stat =
  statPanel.new('Tower Version', datasource='$datasource', colorMode='background', graphMode='none')
  .addTarget(grafana.prometheus.target(queries.awx_system_info, format='table', instant=true))
  + {
    target_publisher:: true,
    options+: {
      reduceOptions: {
        values: false,
        calcs: [
          'lastNotNull',
        ],
        fields: '/^tower_version$/',
      },
    },
    fieldConfig+: {
      defaults+: {
        color: { mode: 'palette-classic' },
      },
    },
  };

local license_type_stat =
  statPanel.new(
    'License Type',
    colorMode='background',
    graphMode='none',
    justifyMode='center',
  )
  {
    target_subscriber:: [queries.awx_system_info],
    options+: {
      reduceOptions: {
        values: false,
        calcs: [
          'lastNotNull',
        ],
        fields: '/^license_type$/',
      },
    },
    fieldConfig+: {
      defaults+: {
        color: { mode: 'fixed', fixedColor: 'blue' },
      },
    },
  };


local license_expiry_stat =
  statPanel.new('License Expiry', datasource='$datasource', colorMode='background', graphMode='none', unit='s')
  .addThresholds([
    {
      color: 'red',
      value: null,
    },
    {
      color: '#EAB839',
      value: 7776000,
    },
    {
      color: 'super-light-green',
      value: 15552000,
    },
    {
      value: 31536000,
      color: 'green',
    },
  ]) +
  {
    target_subscriber:: [queries.awx_system_info],
    options+: {
      reduceOptions: {
        values: false,
        calcs: [
          'lastNotNull',
        ],
        fields: '/^license_expiry$/',
      },
    },
    fieldConfig+: {
      defaults+: {
        color: { mode: 'thresholds' },
      },
    },
    transformations: [
      {
        id: 'labelsToFields',
      },
    ],
  };

local license_total_free_stat =
  statPanel.new('Licenses Free/Total', datasource='$datasource', colorMode='background', graphMode='none', textMode='value', justifyMode='center')
  .addTargets([
    grafana.prometheus.target(queries.license_free),
    grafana.prometheus.target(queries.license_total),
  ])
  .addThresholds([
    {
      color: 'red',
      value: null,
    },
    {
      value: 10,
      color: '#EAB839',
    },
    {
      value: 20,
      color: '#EF843C',
    },
    {
      value: 40,
      color: 'green',
    },
  ])
  .addOverride(
    matcher={
      id: 'byName',
      options: 'awx_license_instance_total{cluster="my-cluster", instance="awx-service.sample-apps.svc.cluster.local:80", job="integrations/awx"}',
    },
    properties=[
      {
        id: 'thresholds',
        value: {
          mode: 'absolute',
          steps: [
            {
              color: 'blue',
              value: null,
            },
          ],
        },
      },
    ]
  );

local templates_by_type_pie =
  pieChartPanel.new('Templates by Type', datasource='$datasource')
  .addTarget(grafana.prometheus.target(queries.job_templates_total, legendFormat='Job'))
  .addTarget(grafana.prometheus.target(queries.workflow_job_templates_total, legendFormat='Workflow Job'))
  + piechartupdate;

local jobs_by_status =
  graphPanel.new(
    'Active Jobs by State',
    datasource='$datasource',
    stack=true,
    description='Total number of jobs in the Running or Pending state, on all AWX instances in the cluster.'
  )
  .addTargets([
    grafana.prometheus.target(queries.running_job_count, legendFormat='Running'),
    grafana.prometheus.target(queries.pending_job_count, legendFormat='Pending'),
  ]);

local job_status_rate =
  graphPanel.new(
    'Job Completion Status Rate',
    datasource='$datasource',
    stack=true,
    description='Rate of job completion status, on all AWX instances in the cluster.'
  )
  .addTarget(grafana.prometheus.target(queries.job_status_rate, legendFormat='{{status}}'));

local jobs_launch_type_rate_graph =
  graphPanel.new(
    'Job Rate by Launch Type',
    datasource='$datasource',
    stack=true,
    description='Rate of new jobs being started, by Launch Type.'
  )
  .addTargets([
    grafana.prometheus.target(queries.job_launch_type_rate, legendFormat='{{launch_type}}'),
  ]);

local jobs_status_rate_graph =
  graphPanel.new(
    'Job Rate by Node and Completion Status',
    datasource='$datasource',
    stack=true,
    description='Rate of new jobs being completed, by their Node and Completion Status.'
  )
  .addTargets([
    grafana.prometheus.target(queries.instance_job_status_rate, legendFormat='{{status}} - {{node}}'),
  ]);

local jobs_by_launch_type_pie =
  pieChartPanel.new('Jobs by Launch Type', datasource='$datasource')
  .addTarget(grafana.prometheus.target(queries.job_launch_type_total, legendFormat='{{launch_type}}'))
  + piechartupdate;

local jobs_by_status_pie =
  pieChartPanel.new('Jobs by Completion Status', datasource='$datasource')
  .addTarget(grafana.prometheus.target(queries.job_status_total, legendFormat='{{status}}'))
  + piechartupdate;

local awx_dashboard =
  dashboard.new(
    'AWX',
    uid=dashboardUid,
    time_from='now-1h',
  ).addTemplates([
    ds_template,
    job_template,
    instance_template,
  ])
  .addPanels([
    row.new('Cluster'),
    tower_ver_stat { gridPos: { h: 3, w: 3, x: 0, y: 7 } },
    license_type_stat { gridPos: { h: 3, w: 3, x: 15, y: 7 } },
    license_expiry_stat { gridPos: { h: 3, w: 3, x: 18, y: 7 } },
    license_total_free_stat { gridPos: { h: 3, w: 3, x: 21, y: 7 } },
    sessions_pie { gridPos: { h: 5, w: 12, x: 6, y: 16 } },
    baseurl_stat { gridPos: { h: 3, w: 12, x: 3, y: 7 } },
    ct.new('$datasource', dashboardUid) { gridPos: { h: 6, w: 24, x: 0, y: 1 } },
    hosts_pie { gridPos: { h: 5, w: 6, x: 0, y: 16 } },
    it.new(
      '$datasource',
      grafana.prometheus.target(queries.instance_info, format='table', instant=true),
      grafana.prometheus.target(queries.instance_cpu, format='table', instant=true),
      grafana.prometheus.target(queries.instance_memory, format='table', instant=true),
      grafana.prometheus.target(queries.instance_consumed_capacity_percent, format='table', instant=true),
    ) { gridPos: { h: 6, w: 24, x: 0, y: 10 } },
    templates_by_type_pie { gridPos: { h: 5, w: 6, x: 18, y: 16 } },
    row.new('Jobs') + { gridPos+: { y: 22 } },
    jobs_by_status { gridPos: { x: 0, y: 23, w: 9, h: 6 } },
    job_status_rate { gridPos: { x: 0, y: 29, w: 9, h: 6 } },
    jobs_launch_type_rate_graph { gridPos: { x: 9, y: 22, w: 9, h: 6 } },
    jobs_status_rate_graph { gridPos: { x: 9, y: 28, w: 9, h: 6 } },
    jobs_by_launch_type_pie { gridPos: { x: 18, y: 22, w: 6, h: 6 } },
    jobs_by_status_pie { gridPos: { x: 18, y: 28, w: 6, h: 6 } },
  ]);

{
  grafanaDashboards+:: {
    'awx.json':
      awx_dashboard
      + target_matchmake(awx_dashboard.panels),
  },
}
