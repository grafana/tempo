{
  new(datasource, info_target, cpu_target, mem_target, consumed_capacity_target):: {
    type: 'table',
    title: 'Cluster Instances',
    targets: [
      info_target,
      cpu_target,
      mem_target,
      consumed_capacity_target,
    ],
    options: {
      showHeader: true,
      footer: {
        show: false,
        reducer: [
          'sum',
        ],
        fields: '',
      },
    },
    fieldConfig: {
      defaults: {
        custom: {
          align: 'auto',
          displayMode: 'auto',
          minWidth: 100,
        },
        thresholds: {
          mode: 'absolute',
          steps: [
            {
              value: null,
              color: 'green',
            },
            {
              value: 80,
              color: 'red',
            },
          ],
        },
        mappings: [],
        color: {
          mode: 'thresholds',
        },
      },
      overrides: [
        {
          matcher: {
            id: 'byName',
            options: 'Enabled',
          },
          properties: [
            {
              id: 'custom.displayMode',
              value: 'color-background',
            },
            {
              id: 'mappings',
              value: [
                {
                  type: 'value',
                  options: {
                    True: {
                      color: 'green',
                      index: 0,
                    },
                    False: {
                      color: 'red',
                      index: 1,
                    },
                  },
                },
              ],
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'Consumed Capacity',
          },
          properties: [
            {
              id: 'custom.displayMode',
              value: 'color-background',
            },
            {
              id: 'thresholds',
              value: {
                mode: 'absolute',
                steps: [
                  {
                    color: 'green',
                    value: null,
                  },
                  {
                    value: 0.4,
                    color: '#EAB839',
                  },
                  {
                    value: 0.6,
                    color: 'orange',
                  },
                  {
                    color: 'red',
                    value: 0.8,
                  },
                ],
              },
            },
            {
              id: 'unit',
              value: 'percentunit',
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'Memory',
          },
          properties: [
            {
              id: 'unit',
              value: 'bytes',
            },
          ],
        },
      ],
    },
    transformations: [
      {
        id: 'merge',
        options: {},
      },
      {
        id: 'organize',
        options: {
          excludeByName: {
            Time: true,
            __name__: true,
            cluster: true,
            managed_by_policy: true,
            instance: true,
            job: true,
            'Value #A': true,
          },
          indexByName: {
            Time: 0,
            __name__: 1,
            cluster: 2,
            hostname: 3,
            enabled: 4,
            managed_by_policy: 5,
            instance: 6,
            instance_uuid: 7,
            job: 8,
            version: 9,
            'Value #A': 10,
            'Value #B': 11,
            'Value #C': 12,
            'Vaule #D': 13,
          },
          renameByName: {
            hostname: 'Hostname',
            enabled: 'Enabled',
            instance_uuid: 'UUID',
            managed_by_policy: 'Managed by Policy',
            version: 'Version',
            'Value #B': 'CPU Cores',
            'Value #C': 'Memory',
            'Value #D': 'Consumed Capacity',
          },
        },
      },
    ],
    datasource: datasource,
  },
}
