{
  wintable(title, datasource):: {
    local s = self,
    type: 'table',
    title: title,
    datasource: datasource,
    _overrides:: [],
    _hiddenColumns:: [],
    _originalNames:: [],
    _newNames:: [],
    _targets:: [],
    _columns:: [],
    addQuery(expression, name, id):: self {
      _targets+: [{
        expr: expression,
        format: 'table',
        legendFormat: name,
        instant: true,
        refId: id,
      }],
    },
    hideColumn(columnName):: self {
      _hiddenColumns+: [columnName],
    },
    renameColumn(originalName, newName):: self {
      _originalNames+: [originalName],
      _newNames+: [newName],
    },
    setColumnUnit(displayName, unit):: self {
      _overrides+: [{
        matcher: {
          id: 'byName',
          options: displayName,
        },
        properties: [
          {
            id: 'unit',
            value: unit,
          },
        ],
      }],
    },
    addColumn(column):: self {
      _columns+: [column],
      _targets+: [{
        expr: [column.expression],
        format: 'table',
        legendFormat: [column.name],
        instant: true,
        refId: [column.id],
      }],
    },
    addThreshold(displayName, steps, mode):: self {
      _overrides+: [{
        matcher: {
          id: 'byName',
          options: std.format('%s', displayName),
        },
        properties: [
          {
            id: 'thresholds',
            value: {
              mode: std.format('%s', mode),
              steps: steps,
            },
          },
          {
            id: 'color',
            value: {
              mode: 'thresholds',
            },
          },
          {
            id: 'custom.displayMode',
            value: 'color-background',
          },

        ],
      }],
    },
    targets: s._targets,
    fieldConfig: {
      defaults: {
        color: {
          mode: 'thresholds',
        },
      },
      overrides: s._overrides,
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
            [hiddenColumn.name]: true
            for hiddenColumn in std.makeArray(std.length(s._hiddenColumns), function(x) { name: s._hiddenColumns[x] })
          },
          renameByName: {
            [nameChange.old]: nameChange.new
            for nameChange in std.makeArray(std.length(s._originalNames), function(x) { old: s._originalNames[x], new: s._newNames[x] })
          },
        },
      },
    ],
  },

  winrow(title, showLegend=false, repeat=null):: {
    _panels:: [],
    addWinPanel(panel):: self {
      _panels+: [panel],
    },

    panels: self._panels,
    collapse: false,
    height: '250px',
    repeatIteration: null,
    repeatRowId: null,
    showTitle: true,
    title: title,
    titleSize: 'h6',
    [if repeat != null then 'repeat']: repeat,
  },

  winstat(
    title,
    format='none',
    description='',
    interval=null,
    height=null,
    datasource=null,
    span=null,
    min_span=null,
    decimals=null,
    valueName='avg',
    valueFontSize='80%',
    prefixFontSize='50%',
    postfixFontSize='50%',
    mappingType=1,
    repeat=null,
    repeatDirection=null,
    prefix='',
    postfix='',
    colors=[
      '#299c46',
      'rgba(237, 129, 40, 0.89)',
      '#d44a3a',
    ],
    colorBackground=false,
    colorValue=false,
    thresholds='',
    valueMaps=[
      {
        value: 'null',
        op: '=',
        text: 'N/A',
      },
    ],
    rangeMaps=[
      {
        from: 'null',
        to: 'null',
        text: 'N/A',
      },
    ],
    transparent=null,
    sparklineFillColor='rgba(31, 118, 189, 0.18)',
    sparklineFull=false,
    sparklineLineColor='rgb(31, 120, 193)',
    sparklineShow=false,
    gaugeShow=false,
    gaugeMinValue=0,
    gaugeMaxValue=100,
    gaugeThresholdMarkers=true,
    gaugeThresholdLabels=false,
    timeFrom=null,
    links=[],
    tableColumn='',
    maxPerRow=null,
    overrides=null,
    unit='s',
  )::
    {
      [if height != null then 'height']: height,
      [if description != '' then 'description']: description,
      [if repeat != null then 'repeat']: repeat,
      [if repeatDirection != null then 'repeatDirection']: repeatDirection,
      [if transparent != null then 'transparent']: transparent,
      [if min_span != null then 'minSpan']: min_span,
      title: title,
      [if span != null then 'span']: span,
      type: 'stat',
      datasource: datasource,
      targets: [
      ],
      links: links,
      [if decimals != null then 'decimals']: decimals,
      maxDataPoints: 100,
      interval: interval,
      cacheTimeout: null,
      format: format,
      prefix: prefix,
      postfix: postfix,
      nullText: null,
      valueMaps: valueMaps,
      [if maxPerRow != null then 'maxPerRow']: maxPerRow,
      nullPointMode: 'connected',
      fieldConfig: {
        defaults: {
          color: {
            mode: 'thresholds',
          },
          mappings: [
            {
              id: 0,
              op: '=',
              text: 'N/A',
              type: 1,
              value: 'null',
            },
          ],
          unit: unit,
        },
        overrides: overrides,
      },
      valueName: valueName,
      prefixFontSize: prefixFontSize,
      valueFontSize: valueFontSize,
      postfixFontSize: postfixFontSize,
      thresholds: thresholds,
      [if timeFrom != null then 'timeFrom']: timeFrom,
      colorBackground: colorBackground,
      colorValue: colorValue,
      colors: colors,
      gauge: {
        show: gaugeShow,
        minValue: gaugeMinValue,
        maxValue: gaugeMaxValue,
        thresholdMarkers: gaugeThresholdMarkers,
        thresholdLabels: gaugeThresholdLabels,
      },
      sparkline: {
        fillColor: sparklineFillColor,
        full: sparklineFull,
        lineColor: sparklineLineColor,
        show: sparklineShow,
      },
      tableColumn: tableColumn,
      _nextTarget:: 0,
      addTarget(target):: self {
        local nextTarget = super._nextTarget,
        _nextTarget: nextTarget + 1,
        targets+: [target { refId: std.char(std.codepoint('A') + nextTarget) }],
      },
    },
  winbargauge(title, thresholdSteps, expr, exprLegend, span=4)::
    {
      [if span != null then 'span']: span,

      datasource: '${prometheus_datasource}',
      fieldConfig: {
        defaults: {
          color: {
            mode: 'thresholds',
          },
          custom: {},
          mappings: [],
          max: 100,
          min: 0,
          thresholds: {
            mode: 'absolute',
            steps: thresholdSteps,
          },
          unit: 'percent',
        },
        overrides: [],
      },
      links: [],
      options: {
        displayMode: 'lcd',
        orientation: 'horizontal',
        reduceOptions: {
          calcs: [
            'lastNotNull',
          ],
          fields: '',
          values: false,
        },
        showUnfilled: true,
      },
      targets: [
        {
          expr: expr,
          instant: false,
          interval: '',
          legendFormat: '{{volume}}',
          refId: 'A',
        },
      ],
      title: title,
      type: 'bargauge',
    },
}
