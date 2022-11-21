local percentErrs(metric, errSelectors) = '100 * sum(rate(%(metric)s{%(errSelectors)s}[1m])) by (instance, job, namespace) / sum(rate(%(metric)s[1m])) by (instance, job, namespace)' % {
  metric: metric,
  errSelectors: errSelectors,
};

local percentErrsWithTotal(metric_errs, metric_total) = '100 * sum(rate(%(metric_errs)s[1m])) by (instance, job, namespace) / sum(rate(%(metric_total)s[1m])) by (instance, job, namespace)' % {
  metric_errs: metric_errs,
  metric_total: metric_total,
};

{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'jaeger_alerts',
        rules: [{
          alert: 'JaegerAgentUDPPacketsBeingDropped',
          expr: 'rate(jaeger_agent_thrift_udp_server_packets_dropped_total[1m]) > 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is dropping {{ printf "%.2f" $value }} UDP packets per second.
            |||,
          },
        }, {
          alert: 'JaegerAgentHTTPServerErrs',
          expr: percentErrsWithTotal('jaeger_agent_http_server_errors_total', 'jaeger_agent_http_server_total') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is experiencing {{ printf "%.2f" $value }}% HTTP errors.
            |||,
          },
        }, {
          alert: 'JaegerClientSpansDropped',
          expr: percentErrs('jaeger_reporter_spans', 'result=~"dropped|err"') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              service {{ $labels.job }} {{ $labels.instance }} is dropping {{ printf "%.2f" $value }}% spans.
            |||,
          },
        }, {
          alert: 'JaegerAgentSpansDropped',
          expr: percentErrsWithTotal('jaeger_agent_reporter_batches_failures_total', 'jaeger_agent_reporter_batches_submitted_total') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              agent {{ $labels.job }} {{ $labels.instance }} is dropping {{ printf "%.2f" $value }}% spans.
            |||,
          },
        }, {
          alert: 'JaegerCollectorQueueNotDraining',
          expr: 'avg_over_time(jaeger_collector_queue_length[10m]) > 1000',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              collector {{ $labels.job }} {{ $labels.instance }} is not able to drain the queue.
            |||,
          },
        }, {
          alert: 'JaegerCollectorDroppingSpans',
          expr: percentErrsWithTotal('jaeger_collector_spans_dropped_total', 'jaeger_collector_spans_received_total') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              collector {{ $labels.job }} {{ $labels.instance }} is dropping {{ printf "%.2f" $value }}% spans.
            |||,
          },
        }, {
          alert: 'JaegerSamplingUpdateFailing',
          expr: percentErrs('jaeger_sampler_queries', 'result="err"') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is failing {{ printf "%.2f" $value }}% in updating sampling policies.
            |||,
          },
        }, {
          alert: 'JaegerCollectorPersistenceSlow',
          expr: 'histogram_quantile(0.99, sum by (le) (rate(jaeger_collector_save_latency_bucket[1m]))) > 0.5',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is slow at persisting spans.
            |||,
          },
        }, {
          alert: 'JaegerThrottlingUpdateFailing',
          expr: percentErrs('jaeger_throttler_updates', 'result="err"') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is failing {{ printf "%.2f" $value }}% in updating throttling policies.
            |||,
          },
        }, {
          alert: 'JaegerQueryReqsFailing',
          expr: percentErrs('jaeger_query_requests_total', 'result="err"') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is seeing {{ printf "%.2f" $value }}% query errors on {{ $labels.operation }}.
            |||,
          },
        }, {
          alert: 'JaegerCassandraWritesFailing',
          expr: percentErrsWithTotal('jaeger_cassandra_errors_total', 'jaeger_cassandra_attempts_total') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is seeing {{ printf "%.2f" $value }}% query errors on {{ $labels.operation }}.
            |||,
          },
        }, {
          alert: 'JaegerCassandraReadsFailing',
          expr: percentErrsWithTotal('jaeger_cassandra_read_errors_total', 'jaeger_cassandra_read_attempts_total') + '> 1',
          'for': '15m',
          labels: {
            severity: 'warning',
          },
          annotations: {
            message: |||
              {{ $labels.job }} {{ $labels.instance }} is seeing {{ printf "%.2f" $value }}% query errors on {{ $labels.operation }}.
            |||,
          },
        }],
      },
    ],
  },
}
