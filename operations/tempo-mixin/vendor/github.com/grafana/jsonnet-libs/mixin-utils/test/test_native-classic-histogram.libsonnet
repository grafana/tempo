local utils = import '../utils.libsonnet';
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';

test.new(std.thisFile)

+ test.case.new(
  name='Quantile defaults',
  test=test.expect.eq(
    actual=utils.ncHistogramQuantile('0.95', 'request_duration_seconds', 'cluster="cluster1", job="job1"'),
    expected={
      classic: 'histogram_quantile(0.95, sum by (le) (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1"}[$__rate_interval])))',
      native: 'histogram_quantile(0.95, sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))',
    },
  )
)
+ test.case.new(
  name='Quantile different groups, interval, multiplier',
  test=test.expect.eq(
    actual=utils.ncHistogramQuantile('0.95', 'request_duration_seconds', 'cluster="cluster1", job="job1"', ['namespace', 'route'], '5m', '42'),
    expected={
      classic: 'histogram_quantile(0.95, sum by (le,namespace,route) (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1"}[5m]))) * 42',
      native: 'histogram_quantile(0.95, sum by (namespace,route) (rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m]))) * 42',
    },
  )
)
+ test.case.new(
  name='Quantile in recording rule with different groups, interval, multiplier, rate',
  test=test.expect.eq(
    actual=utils.ncHistogramQuantile('0.95', 'request_duration_seconds', 'cluster="cluster1", job="job1"', ['namespace', 'route'], '5m', '42', true),
    expected={
      classic: 'histogram_quantile(0.95, sum by (le,namespace,route) (request_duration_seconds_bucket:sum_rate{cluster="cluster1", job="job1"})) * 42',
      native: 'histogram_quantile(0.95, sum by (namespace,route) (request_duration_seconds:sum_rate{cluster="cluster1", job="job1"})) * 42',
    },
  )
)
+ test.case.new(
  name='Quantile different groups, interval, multiplier, offset',
  test=test.expect.eq(
    actual=utils.ncHistogramQuantile('0.95', 'request_duration_seconds', 'cluster="cluster1", job="job1"', ['namespace', 'route'], '5m', '42', offset='1w'),
    expected={
      classic: 'histogram_quantile(0.95, sum by (le,namespace,route) (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1"}[5m] offset 1w))) * 42',
      native: 'histogram_quantile(0.95, sum by (namespace,route) (rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m] offset 1w))) * 42',
    },
  )
)
+ test.case.new(
  name='Quantile in recording rule with different groups, interval, multiplier, rate, offset',
  test=test.expect.eq(
    actual=utils.ncHistogramQuantile('0.95', 'request_duration_seconds', 'cluster="cluster1", job="job1"', ['namespace', 'route'], '5m', '42', true, offset='1w'),
    expected={
      classic: 'histogram_quantile(0.95, sum by (le,namespace,route) (request_duration_seconds_bucket:sum_rate{cluster="cluster1", job="job1"} offset 1w)) * 42',
      native: 'histogram_quantile(0.95, sum by (namespace,route) (request_duration_seconds:sum_rate{cluster="cluster1", job="job1"} offset 1w)) * 42',
    },
  )
)

+ test.case.new(
  name='rate of sum defaults',
  test=test.expect.eq(
    actual=utils.ncHistogramSumRate('request_duration_seconds', 'cluster="cluster1", job="job1"'),
    expected={
      classic: 'rate(request_duration_seconds_sum{cluster="cluster1", job="job1"}[$__rate_interval])',
      native: 'histogram_sum(rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval]))',
    },
  )
)
+ test.case.new(
  name='rate of sum with different interval',
  test=test.expect.eq(
    actual=utils.ncHistogramSumRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '5m'),
    expected={
      classic: 'rate(request_duration_seconds_sum{cluster="cluster1", job="job1"}[5m])',
      native: 'histogram_sum(rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m]))',
    },
  )
)
+ test.case.new(
  name='rate of sum in recording rule with different interval',
  test=test.expect.eq(
    actual=utils.ncHistogramSumRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '5m', true),
    expected={
      classic: 'request_duration_seconds_sum:sum_rate{cluster="cluster1", job="job1"}',
      native: 'histogram_sum(request_duration_seconds:sum_rate{cluster="cluster1", job="job1"})',
    },
  )
)

+ test.case.new(
  name='rate of count defaults',
  test=test.expect.eq(
    actual=utils.ncHistogramCountRate('request_duration_seconds', 'cluster="cluster1", job="job1"'),
    expected={
      classic: 'rate(request_duration_seconds_count{cluster="cluster1", job="job1"}[$__rate_interval])',
      native: 'histogram_count(rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval]))',
    },
  )
)
+ test.case.new(
  name='rate of count with different interval',
  test=test.expect.eq(
    actual=utils.ncHistogramCountRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '5m'),
    expected={
      classic: 'rate(request_duration_seconds_count{cluster="cluster1", job="job1"}[5m])',
      native: 'histogram_count(rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m]))',
    },
  )
)
+ test.case.new(
  name='rate of count in recording rule with different interval',
  test=test.expect.eq(
    actual=utils.ncHistogramCountRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '5m', true),
    expected={
      classic: 'request_duration_seconds_count:sum_rate{cluster="cluster1", job="job1"}',
      native: 'histogram_count(request_duration_seconds:sum_rate{cluster="cluster1", job="job1"})',
    },
  )
)
+ test.case.new(
  name='rate of count with offset',
  test=test.expect.eq(
    actual=utils.ncHistogramCountRate('request_duration_seconds', 'cluster="cluster1", job="job1"', rate_interval='5m', offset='1w'),
    expected={
      classic: 'rate(request_duration_seconds_count{cluster="cluster1", job="job1"}[5m] offset 1w)',
      native: 'histogram_count(rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m] offset 1w))',
    },
  )
)
+ test.case.new(
  name='rate of count with offset and from_recording',
  test=test.expect.eq(
    actual=utils.ncHistogramCountRate('request_duration_seconds', 'cluster="cluster1", job="job1"', rate_interval='5m', offset='1w', from_recording=true),
    expected={
      classic: 'request_duration_seconds_count:sum_rate{cluster="cluster1", job="job1"} offset 1w',
      native: 'histogram_count(request_duration_seconds:sum_rate{cluster="cluster1", job="job1"} offset 1w)',
    },
  )
)

+ test.case.new(
  name='rate of average defaults',
  test=test.expect.eq(
    actual=utils.ncHistogramAverageRate('request_duration_seconds', 'cluster="cluster1", job="job1"'),
    expected={
      classic: 'sum(rate(request_duration_seconds_sum{cluster="cluster1", job="job1"}[$__rate_interval])) /\nsum(rate(request_duration_seconds_count{cluster="cluster1", job="job1"}[$__rate_interval]))\n',
      native: 'sum(histogram_sum(rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval]))) /\nsum(histogram_count(rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))\n',
    },
  )
)
+ test.case.new(
  name='rate of average with different interval, multiplier',
  test=test.expect.eq(
    actual=utils.ncHistogramAverageRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '5m', '42'),
    expected={
      classic: '42 * sum(rate(request_duration_seconds_sum{cluster="cluster1", job="job1"}[5m])) /\nsum(rate(request_duration_seconds_count{cluster="cluster1", job="job1"}[5m]))\n',
      native: '42 * sum(histogram_sum(rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m]))) /\nsum(histogram_count(rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m])))\n',
    },
  )
)
+ test.case.new(
  name='rate of average with sum_by labels',
  test=test.expect.eq(
    actual=utils.ncHistogramAverageRate('request_duration_seconds', 'cluster="cluster1", job="job1"', sum_by=['namespace']),
    expected={
      classic: 'sum by (namespace) (rate(request_duration_seconds_sum{cluster="cluster1", job="job1"}[$__rate_interval])) /\nsum by (namespace) (rate(request_duration_seconds_count{cluster="cluster1", job="job1"}[$__rate_interval]))\n',
      native: 'sum by (namespace) (histogram_sum(rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval]))) /\nsum by (namespace) (histogram_count(rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))\n',
    },
  )
)
+ test.case.new(
  name='rate of average in recording rule with different interval, multiplier',
  test=test.expect.eq(
    actual=utils.ncHistogramAverageRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '5m', '42', true),
    expected={
      classic: '42 * sum(request_duration_seconds_sum:sum_rate{cluster="cluster1", job="job1"}) /\nsum(request_duration_seconds_count:sum_rate{cluster="cluster1", job="job1"})\n',
      native: '42 * sum(histogram_sum(request_duration_seconds:sum_rate{cluster="cluster1", job="job1"})) /\nsum(histogram_count(request_duration_seconds:sum_rate{cluster="cluster1", job="job1"}))\n',
    },
  )
)

+ test.case.new(
  name='histogram sum by defaults',
  test=test.expect.eq(
    actual=utils.ncHistogramSumBy(utils.ncHistogramCountRate('request_duration_seconds_sum', '{cluster="cluster1", job="job1"}')),
    expected={
      classic: 'sum (rate(request_duration_seconds_sum_count{{cluster="cluster1", job="job1"}}[$__rate_interval]))',
      native: 'sum (histogram_count(rate(request_duration_seconds_sum{{cluster="cluster1", job="job1"}}[$__rate_interval])))',
    },
  )
)
+ test.case.new(
  name='histogram sum by with different labels and multiplier',
  test=test.expect.eq(
    actual=utils.ncHistogramSumBy(utils.ncHistogramCountRate('request_duration_seconds_sum', '{cluster="cluster1", job="job1"}'), ['namespace', 'route'], '42'),
    expected={
      classic: 'sum by (namespace, route) (rate(request_duration_seconds_sum_count{{cluster="cluster1", job="job1"}}[$__rate_interval])) * 42',
      native: 'sum by (namespace, route) (histogram_count(rate(request_duration_seconds_sum{{cluster="cluster1", job="job1"}}[$__rate_interval]))) * 42',
    },
  )
)

+ test.case.new(
  name='histogram le rate defaults and le is float',
  test=test.expect.eq(
    actual=utils.ncHistogramLeRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '0.1'),
    expected={
      classic: 'sum (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1", le="0.1"}[$__rate_interval]))',
      native: 'histogram_fraction(0, 0.1, sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))*histogram_count(sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))',
    },
  )
)
+ test.case.new(
  name='histogram le rate defaults and le is whole',
  test=test.expect.eq(
    actual=utils.ncHistogramLeRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '10'),
    expected={
      classic: 'sum (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1", le=~"10|10\\\\.0"}[$__rate_interval]))',
      native: 'histogram_fraction(0, 10.0, sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))*histogram_count(sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))',
    },
  )
)
+ test.case.new(
  name='histogram le rate defaults and le is +Inf',
  test=test.expect.eq(
    actual=utils.ncHistogramLeRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '+Inf'),
    expected={
      classic: 'sum (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1", le="+Inf"}[$__rate_interval]))',
      native: 'histogram_fraction(0, +Inf, sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))*histogram_count(sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))',
    },
  )
)
+ test.case.new(
  name='histogram le rate defaults and le is -Inf',
  test=test.expect.eq(
    actual=utils.ncHistogramLeRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '-Inf'),
    expected={
      classic: 'sum (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1", le="-Inf"}[$__rate_interval]))',
      native: 'histogram_fraction(0, -Inf, sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))*histogram_count(sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))',
    },
  )
)
+ test.case.new(
  name='histogram le rate defaults and le is float with different interval',
  test=test.expect.eq(
    actual=utils.ncHistogramLeRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '0.1', '5m'),
    expected={
      classic: 'sum (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1", le="0.1"}[5m]))',
      native: 'histogram_fraction(0, 0.1, sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m])))*histogram_count(sum (rate(request_duration_seconds{cluster="cluster1", job="job1"}[5m])))',
    },
  )
)
+ test.case.new(
  name='histogram le rate defaults and le is float with sum by',
  test=test.expect.eq(
    actual=utils.ncHistogramLeRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '0.1', sum_by=['cluster', 'namespace']),
    expected={
      classic: 'sum by (cluster, namespace) (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1", le="0.1"}[$__rate_interval]))',
      native: 'histogram_fraction(0, 0.1, sum by (cluster, namespace) (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))*histogram_count(sum by (cluster, namespace) (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval])))',
    },
  )
)
+ test.case.new(
  name='histogram le rate defaults and le is float with sum by and offset',
  test=test.expect.eq(
    actual=utils.ncHistogramLeRate('request_duration_seconds', 'cluster="cluster1", job="job1"', '0.1', sum_by=['cluster', 'namespace'], offset='1w'),
    expected={
      classic: 'sum by (cluster, namespace) (rate(request_duration_seconds_bucket{cluster="cluster1", job="job1", le="0.1"}[$__rate_interval] offset 1w))',
      native: 'histogram_fraction(0, 0.1, sum by (cluster, namespace) (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval] offset 1w)))*histogram_count(sum by (cluster, namespace) (rate(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval] offset 1w)))',
    },
  )
)


+ test.case.new(
  name='commenting histogram query',
  test=test.expect.eq(
    actual=utils.ncHistogramComment({ classic: 'classic_query', native: 'native_query' }, 'comment\n'),
    expected={
      classic: 'comment\nclassic_query\n',
      native: 'comment\nnative_query\n',
    },
  )
)
+ test.case.new(
  name='simple templating',
  test=test.expect.eq(
    actual=utils.ncHistogramApplyTemplate('label_replace(%s, "x", "$1", "y", "(.*)")', { classic: 'classic_query', native: 'native_query' }),
    expected={
      classic: 'label_replace(classic_query, "x", "$1", "y", "(.*)")',
      native: 'label_replace(native_query, "x", "$1", "y", "(.*)")',
    }
  )
)
+ test.case.new(
  name='histogram count increase',
  test=test.expect.eq(
    actual=utils.ncHistogramCountIncrease('request_duration_seconds', 'cluster="cluster1", job="job1"'),
    expected={
      classic: 'increase(request_duration_seconds_count{cluster="cluster1", job="job1"}[$__rate_interval])',
      native: 'histogram_count(increase(request_duration_seconds{cluster="cluster1", job="job1"}[$__rate_interval]))',
    }
  )
)
+ test.case.new(
  name='histogram count increase with rate interval',
  test=test.expect.eq(
    actual=utils.ncHistogramCountIncrease('request_duration_seconds', 'cluster="cluster1", job="job1"', rate_interval='5m'),
    expected={
      classic: 'increase(request_duration_seconds_count{cluster="cluster1", job="job1"}[5m])',
      native: 'histogram_count(increase(request_duration_seconds{cluster="cluster1", job="job1"}[5m]))',
    }
  )
)
+ test.case.new(
  name='showClassicHistogramQuery defaults',
  test=test.expect.eq(
    actual=utils.showClassicHistogramQuery({ classic: 'foo' }),
    expected='(foo) and on() (vector($latency_metrics) == 1)',
  )
)
+ test.case.new(
  name='showClassicHistogramQuery other variable',
  test=test.expect.eq(
    actual=utils.showClassicHistogramQuery({ classic: 'foo' }, dashboard_variable='my_var'),
    expected='(foo) and on() (vector($my_var) == 1)',
  )
)
+ test.case.new(
  name='showClassicHistogramQuery disable',
  test=test.expect.eq(
    actual=utils.showClassicHistogramQuery({ classic: 'foo' }, disable=true),
    expected='(foo) and on() (vector(1) == 1)',
  )
)
+ test.case.new(
  name='showClassicHistogramQuery disable ignore dashboard variable',
  test=test.expect.eq(
    actual=utils.showClassicHistogramQuery({ classic: 'foo' }, dashboard_variable='my_var', disable=true),
    expected='(foo) and on() (vector(1) == 1)',
  )
)
+ test.case.new(
  name='showNativeHistogramQuery defaults',
  test=test.expect.eq(
    actual=utils.showNativeHistogramQuery({ native: 'foo' }),
    expected='(foo) and on() (vector($latency_metrics) == -1)',
  )
)
+ test.case.new(
  name='showNativeHistogramQuery other variable',
  test=test.expect.eq(
    actual=utils.showNativeHistogramQuery({ native: 'foo' }, dashboard_variable='my_var'),
    expected='(foo) and on() (vector($my_var) == -1)',
  )
)
+ test.case.new(
  name='showNativeHistogramQuery disable',
  test=test.expect.eq(
    actual=utils.showNativeHistogramQuery({ native: 'foo' }, disable=true),
    expected='(foo) and on() (vector(1) == -1)',
  )
)
+ test.case.new(
  name='showNativeHistogramQuery disable ignore dashboard variable',
  test=test.expect.eq(
    actual=utils.showNativeHistogramQuery({ native: 'foo' }, dashboard_variable='my_var', disable=true),
    expected='(foo) and on() (vector(1) == -1)',
  )
)
