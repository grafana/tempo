local grafana = import 'github.com/grafana/grafonnet-lib/grafonnet/grafana.libsonnet';
local prometheus = grafana.prometheus;

{
  new(title, description=null)::
    grafana.graphPanel.new(
      title,
      datasource='$datasource',
      description=description
    )
    +
    {
      type: 'timeseries',
      options+: {
        tooltip: {
          mode: 'multi',
        },
      },
      fieldConfig+: {
        defaults+: {
          custom+: {
            lineInterpolation: 'smooth',
            fillOpacity: 0,
            showPoints: 'never',
          },
        },
      },

      addDataLink(title, url):: self {

        fieldConfig+: {
          defaults+: {
            links: [
              {
                title: title,
                url: url,
              },
            ],
          },
        },
      },

      withUnit(unit):: self {

        fieldConfig+: {
          defaults+: {
            unit: unit,
          },
        },
      },

      withFillOpacity(opacity):: self {
        fieldConfig+: {
          defaults+: {
            custom+: {

              fillOpacity: opacity,
            },
          },
        },

      },
    },
}
