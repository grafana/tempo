{
  new(datasource, dashboardUid, span=12):: {
    type: 'table',
    title: 'Historical Instances',
    span: 12,
    transformations: [
      {
        id: 'seriesToColumns',
        options: {
          byField: 'instance',
        },
      },
      {
        id: 'calculateField',
        options: {
          mode: 'binary',
          reduce: {
            reducer: 'sum',
          },
          binary: {
            left: 'Value #B',
            operator: '-',
            right: 'Value #A',
            reducer: 'sum',
          },
          alias: 'delta',
        },
      },
      {
        id: 'filterByValue',
        options: {
          filters: [
            {
              fieldName: 'delta',
              config: {
                id: 'lowerOrEqual',
                options: {
                  value: 0,
                },
              },
            },
          ],
          type: 'exclude',
          match: 'any',
        },
      },
    ],
    datasource: datasource,
    fieldConfig: {
      defaults: {
        custom: {
          align: 'auto',
          displayMode: 'auto',
        },
        thresholds: {
          mode: 'absolute',
          steps: [
            {
              color: 'green',
              value: null,
            },
            {
              color: 'red',
              value: 80,
            },
          ],
        },
        mappings: [],
        color: {
          mode: 'continuous-GrYlRd',
        },
        links: [],
        unit: 'string',
      },
      overrides: [
        {
          matcher: {
            id: 'byRegexp',
            options: '^.+[0-9]+|Value\\s#[B-Z]|delta',
          },
          properties: [
            {
              id: 'custom.hidden',
              value: true,
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'instance',
          },
          properties: [
            {
              id: 'links',
              value: [
                {
                  title: 'View',
                  url: 'd/' + dashboardUid + '/rclone?var-instance=${__data.fields.instance}&from=${__data.fields["Value #A"]}&to=${__data.fields["Value #B"]}',
                },
              ],
            },
            {
              id: 'displayName',
              value: 'Instance',
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'Value #A',
          },
          properties: [
            {
              id: 'displayName',
              value: 'Start Time',
            },
          ],
        },
      ],
    },
    options: {
      showHeader: true,
      footer: {
        show: false,
        reducer: [
          'sum',
        ],
        fields: '',
      },
      frameIndex: 1,
    },
    targets: [],
    _nextTarget:: 0,
    addTarget(target):: self {
      local nextTarget = super._nextTarget,
      _nextTarget: nextTarget + 1,
      targets+: [target { refId: std.char(std.codepoint('A') + nextTarget) }],
    },
    addTargets(targets):: std.foldl(function(p, t) p.addTarget(t), targets, self),
  },
}
