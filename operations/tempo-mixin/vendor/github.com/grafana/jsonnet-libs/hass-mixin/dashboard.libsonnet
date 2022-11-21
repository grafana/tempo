local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');

local base_matcher = 'job=~"$job", instance=~"$instance"';
local entity_matcher = base_matcher + ', entity=~"$entity", friendly_name=~"$friendly_name"';
local available_entity_matcher_decoration = 'and on (entity) entity_available > 0';

local prefix_default = 'homeassistant';
local queries = {
  unsupported_sensor_count: 'count({__name__=~"$prefix\\\\_?sensor_unit_.+", ' + base_matcher + '})',
  unsupported_sensors: '{__name__=~"$prefix\\\\_?sensor_unit_.+", ' + base_matcher + '}',
  entity_count: 'count({__name__=~"$prefix\\\\_?entity_available", ' + base_matcher + '})',
  available_entity_percent: 'count({__name__=~"$prefix\\\\_?entity_available", ' + base_matcher + '} > 0) / ' + queries.entity_count,
  latest_state_change_time: 'bottomk(1, (time() - {__name__=~"$prefix\\\\_?last_updated_time_seconds", ' + base_matcher + '}))',

  input_boolean_state: '{__name__=~"$prefix\\\\_?input_boolean_state", ' + base_matcher + '}',

  battery_percent: '{__name__=~"$prefix\\\\_?battery_level_percent", ' + entity_matcher + '} or {__name__=~"$prefix\\\\_?battery_percent", ' + entity_matcher + '} $Inactive on (entity) label_replace({__name__=~"$prefix\\\\_?entity_available", entity=~".+battery_level.*", ' + entity_matcher + '} == 0, "friendly_name", "$1 (Inactive)", "friendly_name", "(.*)")',
  switch_state: '{__name__=~"$prefix\\\\_?switch_state", ' + entity_matcher + '} $Inactive on (entity) label_replace({__name__=~"$prefix\\\\_?entity_available", domain="switch", ' + entity_matcher + '} == 0, "friendly_name", "$1 (Inactive)", "friendly_name", "(.*)")',
  binary_sensor_state: '{__name__=~"$prefix\\\\_?binary_sensor_state", ' + entity_matcher + '} $Inactive on (entity) label_replace({__name__=~"$prefix\\\\_?entity_available", domain="binary_sensor", ' + entity_matcher + '} == 0, "friendly_name", "$1 (Inactive)", "friendly_name", "(.*)")',
  temperature_c: '{__name__=~"$prefix\\\\_?temperature_c", ' + entity_matcher + '}',
  current_temperature_c: '{__name__=~"$prefix\\\\_?current_temperature_c", ' + entity_matcher + '}',
  light_state: '{__name__=~"$prefix\\\\_?light_state", ' + entity_matcher + '} $Inactive on (entity) label_replace({__name__=~"$prefix\\\\_?entity_available", domain="light", ' + entity_matcher + '} == 0, "friendly_name", "$1 (Inactive)", "friendly_name", "(.*)")',
  lock_state: '{__name__=~"$prefix\\\\_?lock_state", ' + entity_matcher + '} $Inactive on (entity) label_replace({__name__=~"$prefix\\\\_?entity_available", domain="lock", ' + entity_matcher + '} == 0, "friendly_name", "$1 (Inactive)", "friendly_name", "(.*)")',
  climate_action: '{__name__=~"$prefix\\\\_?climate_action", ' + entity_matcher + '} > 0 or label_replace({__name__=~"$prefix\\\\_?humidifier_mode", ' + entity_matcher + '} > 0, "action", "$1", "mode", "(.*)")',
  humidity: '{__name__=~"$prefix\\\\_?humidity_percent", ' + entity_matcher + '}',
  humidity_target: '{__name__=~"$prefix\\\\_?humidifier_target_humidity_percent", ' + entity_matcher + '}',
};

local inverse_colors = ['red', 'yellow', 'green'];

local inactive_overrides = {
  fieldConfig+: {
    overrides: [
      {
        matcher: {
          id: 'byRegexp',
          options: '.+\\(Inactive\\)',
        },
        properties: [
          {
            id: 'color',
            value: {
              mode: 'fixed',
            },
          },
        ],
      },
    ],
  },
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
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local entity_template = grafana.template.new(
  'entity',
  '$datasource',
  'label_values({__name__=~"$prefix\\\\_?last_updated_time_seconds", job=~"$job", instance=~"$instance"}, entity)',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local fname_template = grafana.template.new(
  'friendly_name',
  '$datasource',
  'label_values({__name__=~"$prefix\\\\_?last_updated_time_seconds", job=~"$job", instance=~"$instance"}, friendly_name)',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local prefix_template = {
  current: {
    selected: false,
    text: prefix_default,
    value: prefix_default,
  },
  description: null,
  'error': null,
  hide: 0,
  label: null,
  name: 'prefix',
  options: [
    {
      selected: true,
      text: prefix_default,
      value: prefix_default,
    },
  ],
  query: prefix_default,
  skipUrlSync: false,
  type: 'textbox',
};

local inactive_template = {
  allValue: null,
  current: {
    selected: true,
    text: 'Include',
    value: 'or',
  },
  description: null,
  'error': null,
  hide: 0,
  includeAll: false,
  label: null,
  multi: false,
  name: 'Inactive',
  options: [
    {
      selected: true,
      text: 'Include',
      value: 'or',
    },
    {
      selected: false,
      text: 'Exclude',
      value: 'unless',
    },
  ],
  query: 'Include : or, Exclude : unless',
  queryValue: '',
  skipUrlSync: false,
  type: 'custom',
};

// Panels
local unsupported_sensor_count_panel = grafana.singlestat.new(
  'Unsupported Sensors',
  span=2,
  datasource='$datasource',
  description=|||
    Sensors which are not supported by the Home Assistant Prometheus will be reported
    as `<namespace>_sensor_unit_<unit>`.

    A full list of unsupported sensors can be found in a table at the bottom of this dashboard.

    You can use [component_config](https://www.home-assistant.io/integrations/prometheus/#component_config)
    and [component_config_glob](https://www.home-assistant.io/integrations/prometheus/#component_config_glob),
    combined with `override_metric` in your Home Assistant configuration file to rewrite unsupported sensors to supported values.

    See the [configuration docs](https://www.home-assistant.io/integrations/prometheus/) for more details.
  |||,
)
                                       .addTarget(
  grafana.prometheus.target(queries.unsupported_sensor_count)
);

local entity_count_panel = grafana.singlestat.new(
  'Total Entities',
  span=2,
  datasource='$datasource',
)
                           .addTarget(
  grafana.prometheus.target(queries.entity_count)
);

local entities_available_percent_panel = grafana.singlestat.new(
  'Entities Active',
  format='percentunit',
  gaugeShow=true,
  colors=inverse_colors,
  thresholds='.80,.90',
  span=2,
  datasource='$datasource',
  gaugeMaxValue=1,
)
                                         .addTarget(grafana.prometheus.target(queries.available_entity_percent));

local latest_update_panel = grafana.statPanel.new(
                              'Latest Update',
                              datasource='$datasource',
                            )
                            .addTarget(
                              grafana.prometheus.target(
                                queries.latest_state_change_time,
                                legendFormat='{{friendly_name}}',
                                instant=true,
                              )
                            )
                            + {
                              span: 2,
                              options+: {
                                textMode: 'value_and_name',
                              },
                            };

local input_bool_panel = grafana.statPanel.new(
                           'Input States',
                           colorMode='background',
                           datasource='$datasource',
                         )
                         .addTarget(
                           grafana.prometheus.target(
                             queries.input_boolean_state,
                             legendFormat='{{friendly_name}}',
                             instant=true
                           ),
                         )
                         .addThresholds([
                           { color: 'red', value: 0 },
                           { color: 'green', value: 1 },
                         ])
                         +
                         {
                           span: 2,
                           options+: { textMode: 'name' },
                         };

local climate_panel = grafana.statPanel.new(
                        'Climate',
                        colorMode='background',
                        datasource='$datasource',
                      )
                      .addTarget(
                        grafana.prometheus.target(
                          queries.climate_action,
                          legendFormat='{{friendly_name}} {{action}}',
                          instant=true
                        ),
                      )
                      +
                      {
                        span: 2,
                        options+: { textMode: 'value_and_name' },
                        fieldConfig+: {
                          overrides: [
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*cooling',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '❄',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                    fixedColor: 'blue',
                                  },
                                },
                              ],
                            },
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*heating',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '🔥',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                    fixedColor: 'red',
                                  },
                                },
                              ],
                            },
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*drying',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '🏜',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                    fixedColor: 'super-light-orange',
                                  },
                                },
                              ],
                            },
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*fan',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '💨',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                    fixedColor: 'green',
                                  },
                                },
                              ],
                            },
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*idle',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '💤',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                    fixedColor: 'super-light-yellow',
                                  },
                                },
                              ],
                            },
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*off',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '⛔',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                  },
                                },
                              ],
                            },
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*eco',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '🌏',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                    fixedColor: 'green',
                                  },
                                },
                              ],
                            },
                            {
                              matcher: {
                                id: 'byRegexp',
                                options: '.*home',
                              },
                              properties: [
                                {
                                  id: 'mappings',
                                  value: [
                                    {
                                      type: 1,
                                      from: '',
                                      to: '',
                                      text: '🏠',
                                      value: 1,
                                    },
                                  ],
                                },
                                {
                                  id: 'color',
                                  value: {
                                    mode: 'fixed',
                                    fixedColor: 'yellow',
                                  },
                                },
                              ],
                            },
                          ],
                        },
                      };

local battery_percent_panel = grafana.barGaugePanel.new(
                                'Battery Levels',
                                unit='percent',
                                datasource='$datasource',
                                thresholds=[
                                  { color: 'red', value: 0 },
                                  { color: 'yellow', value: 40 },
                                  { color: 'green', value: 75 },
                                ],
                              )
                              .addTarget(
                                grafana.prometheus.target(
                                  queries.battery_percent,
                                  legendFormat='{{friendly_name}}',
                                  instant=true
                                )
                              )
                              +
                              {
                                options+: {
                                  displayMode: 'lcd',
                                  showUnfilled: true,
                                },
                                span: 4,
                              }
                              + inactive_overrides;

local switch_state_panel = grafana.statPanel.new(
                             'Switch States',
                             colorMode='background',
                             datasource='$datasource',
                             graphMode='none',
                           )
                           .addTarget(
                             grafana.prometheus.target(
                               queries.switch_state,
                               legendFormat='{{friendly_name}}',
                               instant=true
                             ),
                           )
                           .addMappings([
                             {
                               to: '',
                               text: '✔',
                               value: 1,
                               type: 1,
                             },
                             {
                               to: '',
                               text: '❌',
                               value: 0,
                               type: 1,
                             },
                           ])
                           .addThresholds([
                             { color: 'red', value: 0 },
                             { color: 'green', value: 1 },
                           ])
                           +
                           { span: 4, options+: { textMode: 'value_and_name' } } +
                           inactive_overrides;

local binary_sensor_panel = grafana.statPanel.new(
                              'Binary Sensors',
                              colorMode='background',
                              datasource='$datasource',
                            )
                            .addTarget(
                              grafana.prometheus.target(
                                queries.binary_sensor_state,
                                legendFormat='{{friendly_name}}',
                                instant=true
                              ),
                            )
                            .addThresholds([
                              { color: 'red', value: 0 },
                              { color: 'green', value: 1 },
                            ])
                            +
                            {
                              span: 4,
                              options+: { textMode: 'name' },
                            } + inactive_overrides;

local temperature_c_panel = grafana.graphPanel.new(
                              'Temperature',
                              span=6,
                              datasource='$datasource',
                            ) +
                            g.queryPanel(
                              [queries.temperature_c, queries.current_temperature_c],
                              ['{{friendly_name}}', '{{friendly_name}} Current'],
                            ) +
                            {
                              yaxes: g.yaxes('celsius'),
                            };

local humidity_panel = grafana.graphPanel.new(
                         'Humidity',
                         span=6,
                         datasource='$datasource',
                       ) +
                       g.queryPanel(
                         [queries.humidity, queries.humidity_target],
                         ['{{friendly_name}}', '{{friendly_name}} Target'],
                       ) +
                       {
                         yaxes: g.yaxes('percent'),
                       };

local lights_panel = grafana.barGaugePanel.new(
                       'Lights',
                       unit='percent',
                       datasource='$datasource',
                     )
                     .addTarget(
                       grafana.prometheus.target(
                         queries.light_state,
                         legendFormat='{{friendly_name}}',
                         instant=true
                       )
                     )
                     +
                     {
                       options+: {
                         displayMode: 'gradient',
                         showUnfilled: true,
                       },
                       fieldConfig+: {
                         defaults+: {
                           color: {
                             mode: 'continuous-purples',
                           },
                         },
                       },
                       span: 6,
                     }
                     + inactive_overrides;

local lock_state_panel = grafana.statPanel.new(
                           'Locks',
                           colorMode='background',
                           datasource='$datasource',
                           graphMode='none',
                         )
                         .addTarget(
                           grafana.prometheus.target(
                             queries.lock_state,
                             legendFormat='{{friendly_name}}',
                             instant=true
                           ),
                         )
                         .addMappings([
                           {
                             to: '',
                             text: '🔒',
                             value: 1,
                             type: 1,
                           },
                           {
                             to: '',
                             text: '🔓',
                             value: 0,
                             type: 1,
                           },
                         ])
                         .addThresholds([
                           { color: 'red', value: 0 },
                           { color: 'green', value: 1 },
                         ])
                         +
                         {
                           span: 6,
                           options+: { textMode: 'value_and_name' },
                         } + inactive_overrides;

local unsupported_sensors_panel = g.tablePanel(
  [queries.unsupported_sensors],
  {
    entity: { alias: 'Entity' },
    __name__: { alias: 'Metric' },
    'Value #A': { alias: 'Value' },
  }
) + { span: 12, datasource: '$datasource', title: 'Unsupported Sensors' };

// Manifested stuff starts here
{
  grafanaDashboards+:: {
    'hass.json':
      grafana.dashboard.new('Home Assistant', uid='LYCv3YDFD')
      .addTemplates([
        ds_template,
        job_template,
        instance_template,
        entity_template,
        fname_template,
        inactive_template,
        prefix_template,
      ])

      // Overview Row
      .addRow(
        grafana.row.new('Overview')
        // Unsupported Sensors
        .addPanel(unsupported_sensor_count_panel)

        // Total Entities
        .addPanel(entity_count_panel)

        // Available Entities
        .addPanel(entities_available_percent_panel)

        // Latest Update
        .addPanel(latest_update_panel)

        // Input States
        .addPanel(input_bool_panel)

        // Climate
        .addPanel(climate_panel)
      )

      // Switch Row
      .addRow(
        grafana.row.new('States')

        // Switch States
        .addPanel(switch_state_panel)

        // Binary Sensors
        .addPanel(binary_sensor_panel)

        // Battery Level
        .addPanel(battery_percent_panel)

        // Lights
        .addPanel(lights_panel)

        // Locks
        .addPanel(lock_state_panel)
      )

      // Sensors Row
      .addRow(
        grafana.row.new('Sensors')

        // Temperature C
        .addPanel(temperature_c_panel)

        // Humidity
        .addPanel(humidity_panel)

        // Unsuported Sensors
        .addPanel(unsupported_sensors_panel)
      ) +
      { graphTooltip: 2 },  // Shared tooltip. When you hover over a graph, the same time is selected on all graphs, and tooltip is shown. Set to 1 to only share crosshair
  },
}
