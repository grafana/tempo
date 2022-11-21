{
  timeSeriesOverride(
    unit='none'
  ):: {
    type: 'timeseries',
    fieldConfig: {
      defaults: {
        unit: unit,
      },
    },
  },
}
