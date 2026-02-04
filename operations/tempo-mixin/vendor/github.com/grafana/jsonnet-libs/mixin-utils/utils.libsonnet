local g = import 'grafana-builder/grafana.libsonnet';

{
  isNativeClassicQuery(query):: if std.isObject(query) then std.objectHas(query, 'native') && std.objectHas(query, 'classic') else false,

  // The ncHistogramQuantile (native classic histogram quantile) function is
  // used to calculate histogram quantiles from native histograms or classic
  // histograms. Metric name should be provided without _bucket suffix.
  // If from_recording is true, the function will assume :sum_rate metric
  // suffix and no rate needed.
  ncHistogramQuantile(percentile, metric, selector, sum_by=[], rate_interval='$__rate_interval', multiplier='', from_recording=false, offset='')::
    local offsetStr = if offset == '' then '' else ' offset ' + offset;
    local classicSumBy = if std.length(sum_by) > 0 then ' by (%(lbls)s) ' % { lbls: std.join(',', ['le'] + sum_by) } else ' by (le) ';
    local nativeSumBy = if std.length(sum_by) > 0 then ' by (%(lbls)s) ' % { lbls: std.join(',', sum_by) } else ' ';
    local multiplierStr = if multiplier == '' then '' else ' * %s' % multiplier;
    local rateOpen = if from_recording then '' else 'rate(';
    local rateClose = if from_recording then offsetStr else '[%s]%s)' % [rate_interval, offsetStr];
    {
      classic: 'histogram_quantile(%(percentile)s, sum%(classicSumBy)s(%(rateOpen)s%(metric)s_bucket%(suffix)s{%(selector)s}%(rateClose)s))%(multiplierStr)s' % {
        classicSumBy: classicSumBy,
        metric: metric,
        multiplierStr: multiplierStr,
        percentile: percentile,
        rateInterval: rate_interval,
        rateOpen: rateOpen,
        rateClose: rateClose,
        selector: selector,
        suffix: if from_recording then ':sum_rate' else '',
      },
      native: 'histogram_quantile(%(percentile)s, sum%(nativeSumBy)s(%(rateOpen)s%(metric)s%(suffix)s{%(selector)s}%(rateClose)s))%(multiplierStr)s' % {
        metric: metric,
        multiplierStr: multiplierStr,
        nativeSumBy: nativeSumBy,
        percentile: percentile,
        rateInterval: rate_interval,
        rateOpen: rateOpen,
        rateClose: rateClose,
        selector: selector,
        suffix: if from_recording then ':sum_rate' else '',
      },
    },

  // The ncHistogramSumRate (native classic histogram sum rate) function is
  // used to calculate the histogram rate of the sum from native histograms or
  // classic histograms. Metric name should be provided without _sum suffix.
  // If from_recording is true, the function will assume :sum_rate metric
  // suffix and no rate needed.
  ncHistogramSumRate(metric, selector, rate_interval='$__rate_interval', from_recording=false, offset='')::
    $.ncHistogramChange('sum', 'rate', metric, selector, rate_interval, from_recording, offset),

  // The ncHistogramCountRate (native classic histogram count rate) function is
  // used to calculate the histogram rate of count from native histograms or
  // classic histograms. Metric name should be provided without _count suffix.
  // If from_recording is true, the function will assume :sum_rate metric
  // suffix and no rate needed.
  ncHistogramCountRate(metric, selector, rate_interval='$__rate_interval', from_recording=false, offset='')::
    $.ncHistogramChange('count', 'rate', metric, selector, rate_interval, from_recording, offset),

  // The ncHistogramCountIncrease (native classic histogram count rate) function is
  // used to calculate the histogram increase of count from native histograms or
  // classic histograms. Metric name should be provided without _count suffix.
  ncHistogramCountIncrease(metric, selector, rate_interval='$__rate_interval', offset='')::
    $.ncHistogramChange('count', 'increase', metric, selector, rate_interval, false, offset),

  // ncHistogramChange is a helper function to generate queries for either
  // histogram sum or count changes over time using the specified function
  // (e.g., rate, increase). Metric name should be provided without _sum or
  // _count suffix.
  ncHistogramChange(sum_or_count, func_name, metric, selector, rate_interval, from_recording, offset)::
    local offsetStr = if offset == '' then '' else ' offset ' + offset;
    local funcOpen = if from_recording then '' else '%s(' % func_name;
    local funcClose = if from_recording then offsetStr else '[%s]%s)' % [rate_interval, offsetStr];
    local suffix = if from_recording then ':sum_%s' % func_name else '';
    {
      classic: '%(funcOpen)s%(metric)s_%(sum_or_count)s%(suffix)s{%(selector)s}%(funcClose)s' % {
        metric: metric,
        sum_or_count: sum_or_count,
        funcOpen: funcOpen,
        funcClose: funcClose,
        selector: selector,
        suffix: suffix,
      },
      native: 'histogram_%(sum_or_count)s(%(funcOpen)s%(metric)s%(suffix)s{%(selector)s}%(funcClose)s)' % {
        metric: metric,
        sum_or_count: sum_or_count,
        funcOpen: funcOpen,
        funcClose: funcClose,
        selector: selector,
        suffix: suffix,
      },
    },

  // TODO(krajorama) Switch to histogram_avg function for native histograms later.
  // ncHistogramAverageRate (native classic histogram average rate) function is
  // used to calculate the histogram average rate from native histograms or
  // classic histograms.
  // If from_recording is true, the function will assume :sum_rate metric
  // suffix and no rate needed.
  ncHistogramAverageRate(metric, selector, rate_interval='$__rate_interval', multiplier='', from_recording=false, sum_by=[], offset='')::
    local sumBy = if std.length(sum_by) > 0 then ' by (%s) ' % std.join(', ', sum_by) else '';
    local multiplierStr = if multiplier == '' then '' else '%s * ' % multiplier;
    {
      classic: |||
        %(multiplier)ssum%(sumBy)s(%(sumMetricQuery)s) /
        sum%(sumBy)s(%(countMetricQuery)s)
      ||| % {
        sumMetricQuery: $.ncHistogramSumRate(metric, selector, rate_interval, from_recording, offset).classic,
        countMetricQuery: $.ncHistogramCountRate(metric, selector, rate_interval, from_recording, offset).classic,
        multiplier: multiplierStr,
        sumBy: sumBy,
      },
      native: |||
        %(multiplier)ssum%(sumBy)s(%(sumMetricQuery)s) /
        sum%(sumBy)s(%(countMetricQuery)s)
      ||| % {
        sumMetricQuery: $.ncHistogramSumRate(metric, selector, rate_interval, from_recording, offset).native,
        countMetricQuery: $.ncHistogramCountRate(metric, selector, rate_interval, from_recording, offset).native,
        multiplier: multiplierStr,
        sumBy: sumBy,
      },
    },

  // ncHistogramSumBy (native classic histogram sum by) function is used to
  // generate a query that sums the results of a subquery by the given labels.
  // The function can be used with native histograms or classic histograms.
  ncHistogramSumBy(query, sum_by=[], multiplier='')::
    local sumBy = if std.length(sum_by) > 0 then ' by (%(lbls)s) ' % { lbls: std.join(', ', sum_by) } else ' ';
    local multiplierStr = if multiplier == '' then '' else ' * %s' % multiplier;
    {
      classic: 'sum%(sumBy)s(%(query)s)%(multiplierStr)s' % {
        multiplierStr: multiplierStr,
        query: query.classic,
        sumBy: sumBy,
      },
      native: 'sum%(sumBy)s(%(query)s)%(multiplierStr)s' % {
        multiplierStr: multiplierStr,
        query: query.native,
        sumBy: sumBy,
      },
    },

  // ncHistogramLeRate (native classic histogram le rate) calculates the rate
  // of requests that have a value less than or equal to the given "le" value.
  // The "le" value matcher for classic histograms can handle both Prometheus
  // or OpenMetrics formats, where whole numbers may or may not have ".0" at
  // the end.
  ncHistogramLeRate(metric, selector, le, rate_interval='$__rate_interval', sum_by=[], offset='')::
    local sumBy = if std.length(sum_by) > 0 then ' by (%(lbls)s) ' % { lbls: std.join(', ', sum_by) } else ' ';
    local offsetStr = if offset == '' then '' else ' offset ' + offset;
    local isWholeNumber(str) = str != '' && std.foldl(function(acc, c) acc && (c == '0' || c == '1' || c == '2' || c == '3' || c == '4' || c == '5' || c == '6' || c == '7' || c == '8' || c == '9'), std.stringChars(str), true);
    {
      native: 'histogram_fraction(0, %(le)s, sum%(sumBy)s(rate(%(metric)s{%(selector)s}[%(rateInterval)s]%(offset)s)))*histogram_count(sum%(sumBy)s(rate(%(metric)s{%(selector)s}[%(rateInterval)s]%(offset)s)))' % {
        le: if isWholeNumber(le) then le + '.0' else le,  // Treated as float number.
        metric: metric,
        offset: offsetStr,
        rateInterval: rate_interval,
        selector: selector,
        sumBy: sumBy,
      },
      classic: 'sum%(sumBy)s(rate(%(metric)s_bucket{%(selector)s, %(le)s}[%(rateInterval)s]%(offset)s))' % {
        // le is treated as string, thus it needs to account for Prometheus text format not having '.0', but OpenMetrics having it.
        // Also the resulting string in yaml is stored directly, so the \\ needs to be escaped to \\\\.
        le: if isWholeNumber(le) then 'le=~"%(le)s|%(le)s\\\\.0"' % { le: le } else 'le="%s"' % le,
        metric: metric,
        offset: offsetStr,
        rateInterval: rate_interval,
        selector: selector,
        sumBy: sumBy,
      },
    },

  // ncHistogramApplyTemplate (native classic histogram template applier)
  // Takes a template like 'label_replace(%s, "x", "$1", "y", ".*")'
  // with a single substitution and applies to both the classic and native
  // histogram query.
  ncHistogramApplyTemplate(template, query):: {
    assert $.isNativeClassicQuery(query),
    native: template % query.native,
    classic: template % query.classic,
  },

  // ncHistogramComment (native classic histogram comment) helps attach
  // comments to the query and also keep multiline strings where applicable.
  ncHistogramComment(query, comment):: {
    native: |||
      %s%s
    ||| % [comment, query.native],
    classic: |||
      %s%s
    ||| % [comment, query.classic],
  },

  // showClassicHistogramQuery wraps a query defined as map {classic: q, native: q}, and compares the classic query
  // to dashboard variable which should take -1 or +1 as values in order to hide or show the classic query.
  // If "disable" is true it ignores the dashboard variable and shows the classic query.
  showClassicHistogramQuery(query, dashboard_variable='latency_metrics', disable=false)::
    local testAgainst = if disable then '1' else '$%s' % dashboard_variable;
    '(%s) and on() (vector(%s) == 1)' % [query.classic, testAgainst],

  // showNativeHistogramQuery wraps a query defined as map {classic: q, native: q}, and compares the native query
  // to dashboard variable which should take -1 or +1 as values in order to show or hide the native query.
  // If "disable" is true it ignores the dashboard variable and hides the native query.
  showNativeHistogramQuery(query, dashboard_variable='latency_metrics', disable=false)::
    local testAgainst = if disable then '1' else '$%s' % dashboard_variable;
    '(%s) and on() (vector(%s) == -1)' % [query.native, testAgainst],

  histogramRules(metric, labels, interval='1m', record_native=false)::
    local vars = {
      metric: metric,
      labels_underscore: std.join('_', labels),
      labels_comma: std.join(', ', labels),
      interval: interval,
    };
    [
      {
        record: '%(labels_underscore)s:%(metric)s:99quantile' % vars,
        expr: 'histogram_quantile(0.99, sum(rate(%(metric)s_bucket[%(interval)s])) by (le, %(labels_comma)s))' % vars,
      },
      {
        record: '%(labels_underscore)s:%(metric)s:50quantile' % vars,
        expr: 'histogram_quantile(0.50, sum(rate(%(metric)s_bucket[%(interval)s])) by (le, %(labels_comma)s))' % vars,
      },
      {
        record: '%(labels_underscore)s:%(metric)s:avg' % vars,
        expr: 'sum(rate(%(metric)s_sum[%(interval)s])) by (%(labels_comma)s) / sum(rate(%(metric)s_count[%(interval)s])) by (%(labels_comma)s)' % vars,
      },
      {
        record: '%(labels_underscore)s:%(metric)s_bucket:sum_rate' % vars,
        expr: 'sum(rate(%(metric)s_bucket[%(interval)s])) by (le, %(labels_comma)s)' % vars,
      },
      {
        record: '%(labels_underscore)s:%(metric)s_sum:sum_rate' % vars,
        expr: 'sum(rate(%(metric)s_sum[%(interval)s])) by (%(labels_comma)s)' % vars,
      },
      {
        record: '%(labels_underscore)s:%(metric)s_count:sum_rate' % vars,
        expr: 'sum(rate(%(metric)s_count[%(interval)s])) by (%(labels_comma)s)' % vars,
      },
    ] + if record_native then [
      // Native histogram rule, sum_rate contains the following information:
      // - rate of sum,
      // - rate of count,
      // - rate of sum/count aka average,
      // - rate of buckets,
      // - implicitly the quantile information.
      {
        record: '%(labels_underscore)s:%(metric)s:sum_rate' % vars,
        expr: 'sum(rate(%(metric)s[%(interval)s])) by (%(labels_comma)s)' % vars,
      },
    ] else [],


  // latencyRecordingRulePanel - build a latency panel for a recording rule.
  // - metric: the base metric name (middle part of recording rule name)
  // - selectors: list of selectors which will be added to first part of
  //   recording rule name, and to the query selector itself.
  // - extra_selectors (optional): list of selectors which will be added to the
  //   query selector, but not to the beginnig of the recording rule name.
  //   Useful for external labels.
  // - multiplier (optional): assumes results are in seconds, will multiply
  //   by 1e3 to get ms.  Can be turned off.
  // - sum_by (optional): additional labels to use in the sum by clause, will also be used in the legend
  latencyRecordingRulePanel(metric, selectors, extra_selectors=[], multiplier='1e3', sum_by=[])::
    local labels = std.join('_', [matcher.label for matcher in selectors]);
    local selectorStr = $.toPrometheusSelector(selectors + extra_selectors);
    local sb = ['le'];
    local legend = std.join('', ['{{ %(lb)s }} ' % lb for lb in sum_by]);
    local sumBy = if std.length(sum_by) > 0 then ' by (%(lbls)s) ' % { lbls: std.join(',', sum_by) } else '';
    local sumByHisto = std.join(',', sb + sum_by);
    {
      nullPointMode: 'null as zero',
      yaxes: g.yaxes('ms'),
      targets: [
        {
          expr: 'histogram_quantile(0.99, sum by (%(sumBy)s) (%(labels)s:%(metric)s_bucket:sum_rate%(selector)s)) * %(multiplier)s' % {
            labels: labels,
            metric: metric,
            selector: selectorStr,
            multiplier: multiplier,
            sumBy: sumByHisto,
          },
          format: 'time_series',
          legendFormat: '%(legend)s99th percentile' % legend,
          refId: 'A',
        },
        {
          expr: 'histogram_quantile(0.50, sum by (%(sumBy)s) (%(labels)s:%(metric)s_bucket:sum_rate%(selector)s)) * %(multiplier)s' % {
            labels: labels,
            metric: metric,
            selector: selectorStr,
            multiplier: multiplier,
            sumBy: sumByHisto,
          },
          format: 'time_series',
          legendFormat: '%(legend)s50th percentile' % legend,
          refId: 'B',
        },
        {
          expr: '%(multiplier)s * sum(%(labels)s:%(metric)s_sum:sum_rate%(selector)s)%(sumBy)s / sum(%(labels)s:%(metric)s_count:sum_rate%(selector)s)%(sumBy)s' % {
            labels: labels,
            metric: metric,
            selector: selectorStr,
            multiplier: multiplier,
            sumBy: sumBy,
          },
          format: 'time_series',
          legendFormat: '%(legend)sAverage' % legend,
          refId: 'C',
        },
      ],
    },

  selector:: {
    eq(label, value):: { label: label, op: '=', value: value },
    neq(label, value):: { label: label, op: '!=', value: value },
    re(label, value):: { label: label, op: '=~', value: value },
    nre(label, value):: { label: label, op: '!~', value: value },

    // Use with latencyRecordingRulePanel to get the label in the metric name
    // but not in the selector.
    noop(label):: { label: label, op: 'nop' },
  },

  // latencyRecordingRulePanelNativeHistogram - build a latency panel for a recording rule.
  // - metric: the base metric name (middle part of recording rule name)
  // - selectors: list of selectors which will be added to first part of
  //   recording rule name, and to the query selector itself.
  // - extra_selectors (optional): list of selectors which will be added to the
  //   query selector, but not to the beginnig of the recording rule name.
  //   Useful for external labels.
  // - multiplier (optional): assumes results are in seconds, will multiply
  //   by 1e3 to get ms.  Can be turned off.
  // - sum_by (optional): additional labels to use in the sum by clause, will also be used in the legend
  latencyRecordingRulePanelNativeHistogram(metric, selectors, extra_selectors=[], multiplier='1e3', sum_by=[])::
    local labels = std.join('_', [matcher.label for matcher in selectors]);
    local legend = std.join('', ['{{ %(lb)s }} ' % lb for lb in sum_by]);
    local metricStr = '%(labels)s:%(metric)s' % { labels: labels, metric: metric };
    local selectorStr = $.toPrometheusSelectorNaked(selectors + extra_selectors);
    {
      nullPointMode: 'null as zero',
      yaxes: g.yaxes('ms'),
      targets: [
        {
          expr: $.showClassicHistogramQuery($.ncHistogramQuantile('0.99', metricStr, selectorStr, sum_by=sum_by, multiplier=multiplier, from_recording=true)),
          format: 'time_series',
          legendFormat: '%(legend)s99th percentile' % legend,
          refId: 'A_classic',
        },
        {
          expr: $.showNativeHistogramQuery($.ncHistogramQuantile('0.99', metricStr, selectorStr, sum_by=sum_by, multiplier=multiplier, from_recording=true)),
          format: 'time_series',
          legendFormat: '%(legend)s99th percentile' % legend,
          refId: 'A_native',
        },
        {
          expr: $.showClassicHistogramQuery($.ncHistogramQuantile('0.50', metricStr, selectorStr, sum_by=sum_by, multiplier=multiplier, from_recording=true)),
          format: 'time_series',
          legendFormat: '%(legend)s50th percentile' % legend,
          refId: 'B_classic',
        },
        {
          expr: $.showNativeHistogramQuery($.ncHistogramQuantile('0.50', metricStr, selectorStr, sum_by=sum_by, multiplier=multiplier, from_recording=true)),
          format: 'time_series',
          legendFormat: '%(legend)s50th percentile' % legend,
          refId: 'B_native',
        },
        {
          expr: $.showClassicHistogramQuery($.ncHistogramAverageRate(metricStr, selectorStr, multiplier=multiplier, from_recording=true)),
          format: 'time_series',
          legendFormat: '%(legend)sAverage' % legend,
          refId: 'C_classic',
        },
        {
          expr: $.showNativeHistogramQuery($.ncHistogramAverageRate(metricStr, selectorStr, multiplier=multiplier, from_recording=true)),
          format: 'time_series',
          legendFormat: '%(legend)sAverage' % legend,
          refId: 'C_native',
        },
      ],
    },

  toPrometheusSelectorNaked(selector)::
    local pairs = [
      '%(label)s%(op)s"%(value)s"' % matcher
      for matcher in std.filter(function(matcher) matcher.op != 'nop', selector)
    ];
    '%s' % std.join(', ', pairs),

  toPrometheusSelector(selector):: '{%s}' % $.toPrometheusSelectorNaked(selector),

  // withRunbookURL - Add/Override the runbook_url annotations for all alerts inside a list of rule groups.
  // - url_format: an URL format for the runbook, the alert name will be substituted in the URL.
  // - groups: the list of rule groups containing alerts.
  // - annotation_key: the key to use for the annotation whose value will be the formatted runbook URL.
  withRunbookURL(url_format, groups, annotation_key='runbook_url')::
    local update_rule(rule) =
      if std.objectHas(rule, 'alert')
      then rule {
        annotations+: {
          [annotation_key]: url_format % rule.alert,
        },
      }
      else rule;
    [
      group {
        rules: [
          update_rule(alert)
          for alert in group.rules
        ],
      }
      for group in groups
    ],

  // withLabels adds custom labels to all alert rules in a group or groups
  // Parameters:
  //   labels: A map of label names to values to add to each alert
  //   groups: The alert groups to modify
  //   filter_func: Optional function that returns true for alerts that should be modified
  withLabels(labels, groups, filter_func=null)::
    local defaultFilter = function(rule) true;
    local filterToUse = if filter_func != null then filter_func else defaultFilter;

    std.map(
      function(group)
        group {
          rules: std.map(
            function(rule)
              if std.objectHas(rule, 'alert') && filterToUse(rule)
              then rule {
                labels+: labels,
              }
              else rule,
            group.rules
          ),
        },
      groups
    ),

  removeRuleGroup(groupName):: {
    groups: std.filter(function(group) group.name != groupName, super.groups),
  },

  removeAlertRuleGroup(ruleName):: {
    prometheusAlerts+: $.removeRuleGroup(ruleName),
  },

  removeRecordingRuleGroup(ruleName):: {
    prometheusRules+: $.removeRuleGroup(ruleName),
  },

  overrideAlerts(overrides):: {
    local overrideRule(rule) =
      if 'alert' in rule && std.objectHas(overrides, rule.alert)
      then rule + overrides[rule.alert]
      else rule,
    local overrideInGroup(group) = group { rules: std.map(overrideRule, super.rules) },
    prometheusAlerts+: {
      groups: std.map(overrideInGroup, super.groups),
    },
  },

  removeAlerts(alerts):: {
    local alertNames =
      if std.isObject(alerts)
      then std.objectFields(alerts)
      else alerts,
    local removeRule(rule) = !std.member(alertNames, std.get(rule, 'alert', '')),
    local removeInGroup(group) = group { rules: std.filter(removeRule, super.rules) },
    prometheusAlerts+: {
      groups: std.map(removeInGroup, super.groups),
    },
  },
}
