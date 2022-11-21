local grafana = import 'github.com/grafana/grafonnet-lib/grafonnet/grafana.libsonnet';

{
  new(q1, q2, q3)::
    {
      datasource: {
        type: 'loki',
        uid: '${loki_datasource}',
      },
      fieldConfig: {
        defaults: {
          color: {
            mode: 'fixed',
          },
          custom: {
            axisLabel: '',
            axisPlacement: 'auto',
            axisSoftMin: 0,
            fillOpacity: 50,
            gradientMode: 'none',
            hideFrom: {
              legend: false,
              tooltip: false,
              viz: false,
            },
            lineWidth: 1,
            scaleDistribution: {
              type: 'linear',
            },
          },
          mappings: [],
          thresholds: {
            mode: 'absolute',
            steps: [
              {
                color: 'green',
                value: null,
              },
            ],
          },
          unit: 'short',
        },
        overrides: [
          {
            matcher: {
              id: 'byFrameRefID',
              options: 'A',
            },
            properties: [
              {
                id: 'displayName',
                value: 'Lines',
              },
              {
                id: 'color',
                value: {
                  fixedColor: 'super-light-blue',
                  mode: 'fixed',
                },
              },
            ],
          },
          {
            matcher: {
              id: 'byFrameRefID',
              options: 'B',
            },
            properties: [
              {
                id: 'displayName',
                value: 'Warnings',
              },
              {
                id: 'color',
                value: {
                  fixedColor: 'orange',
                  mode: 'fixed',
                },
              },
            ],
          },
          {
            matcher: {
              id: 'byFrameRefID',
              options: 'C',
            },
            properties: [
              {
                id: 'displayName',
                value: 'Errors',
              },
              {
                id: 'color',
                value: {
                  fixedColor: 'red',
                  mode: 'fixed',
                },
              },
            ],
          },
        ],
      },
      maxDataPoints: 25,
      interval: '10s',
      options: {
        barRadius: 0.25,
        barWidth: 0.7,
        groupWidth: 0.5,
        legend: {
          calcs: [],
          displayMode: 'list',
          placement: 'bottom',
        },
        orientation: 'auto',
        showValue: 'never',
        stacking: 'none',
        tooltip: {
          mode: 'multi',
          sort: 'none',
        },
        xTickLabelRotation: 0,
        xTickLabelSpacing: 100,
      },
      targets: [
        {
          datasource: {
            type: 'loki',
            uid: '${loki_datasource}',
          },
          expr: q1,
          refId: 'A',
        },
        {
          datasource: {
            type: 'loki',
            uid: '${loki_datasource}',
          },
          expr: q2,
          hide: false,
          refId: 'B',
        },
        {
          datasource: {
            type: 'loki',
            uid: '${loki_datasource}',
          },
          expr: q3,
          hide: false,
          refId: 'C',
        },
      ],
      title: 'Historical Logs / Warnings / Errors',
      type: 'barchart',
    },
}
