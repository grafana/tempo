{
  timeSeriesOverride(
    unit='none',
    fillOpacity=0,
    lineInterpolation='linear',
    showPoints='auto',
  ):: {
    type: 'timeseries',

    fieldConfig: {
      defaults: {
        unit: unit,
        custom: {
          fillOpacity: fillOpacity,
          lineInterpolation: lineInterpolation,
          showPoints: showPoints,
        },
      },
    },
  },
}
