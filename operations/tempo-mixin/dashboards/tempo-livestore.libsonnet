local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';

// Livestore is mostly diagnosed by zone. The live dashboard uses Grafana's v2
// RowsLayout/GridLayout model; the mixin still emits the classic row model, so
// this keeps the panels and queries equivalent while using the existing helpers.
dashboard_utils {
  grafanaDashboards+: {
    'tempo-livestore.json':
      local liveStoreContainer = $._config.jobs.live_store;
      local liveStorePodRegex = liveStoreContainer + '-zone-.*';
      // Match the source dashboard label order to keep generated query diffs small.
      local namespaceMatcher = 'namespace=~"$namespace", ' + $._config.per_cluster_label + '=~"$cluster"';
      local liveStoreMatcher = namespaceMatcher + ', pod=~"$pod", container="%s"' % liveStoreContainer;
      local liveStoreTenantMatcher = namespaceMatcher + ', pod=~"$pod", tenant=~"$tenant", container="%s"' % liveStoreContainer;
      local liveStoreTenantAfterMatcher = liveStoreMatcher + ', tenant=~"$tenant"';
      local liveStoreIngestMatcher = namespaceMatcher + ', pod=~"$pod", group=~"live-store.*", container="%s"' % liveStoreContainer;
      local zoneFromPod(expr) = 'label_replace(%s, "zone", "$1", "pod", "(%s-[^-]+)-.*")' % [expr, liveStoreContainer + '-zone'];
      local rate(selector) = 'rate(%s[$__rate_interval])' % selector;
      local metric(metricName, matcher=liveStoreMatcher, extra='') = '%s{%s%s}' % [metricName, matcher, extra];
      local metricRate(metricName, matcher=liveStoreMatcher, extra='') = rate(metric(metricName, matcher, extra));
      local zoneRate(metricName, matcher=liveStoreMatcher, extra='') = zoneFromPod(metricRate(metricName, matcher, extra));
      local zoneMetric(metricName, matcher=liveStoreMatcher, extra='') = zoneFromPod(metric(metricName, matcher, extra));
      local queryRoutes = '/tempopb.Querier/.*|/tempopb.Metrics/.*';
      local readRequestMatcher = liveStoreMatcher + ', method="gRPC", route=~"%s"' % queryRoutes;
      local liveStoreAndWarpstreamScaledObjects = 'live-store|warpstream-agent-read|warpstream-agent-write|warpstream-agent-jobs';
      local liveStoreAndWarpstreamHpaRegex = 'keda-hpa-(%s)' % liveStoreAndWarpstreamScaledObjects;

      local row(title, spans=null, collapsed=false) = g.row(title) {
        collapse: collapsed,
        panels:
          local n = std.length(self._panels);
          [
            self._panels[i] {
              span: if spans == null then std.floor(12 / n) else spans[i],
            }
            for i in std.range(0, n - 1)
          ],
      };

      local timeseriesPanel(title, description, queries, legends, unit='short') =
        $.panel(title) +
        $.panelDescription(title, description) +
        $.queryPanel(queries, legends) +
        {
          fieldConfig+: {
            defaults+: {
              decimals: 2,
              unit: unit,
              custom+: {
                fillOpacity: 10,
                showPoints: 'auto',
              },
            },
          },
          options+: {
            legend+: {
              calcs: ['lastNotNull'],
              displayMode: 'table',
              placement: 'bottom',
            },
            tooltip+: {
              mode: 'multi',
              sort: 'desc',
            },
          },
        };

      local statPanel(
        title,
        description,
        queries,
        legends,
        unit='short',
        decimals=1,
        thresholds=[{ color: 'green', value: 0 }]
            ) =
        $.panel(title) +
        $.panelDescription(title, description) +
        $.newStatPanel(queries, legends, unit=unit, decimals=decimals, thresholds=thresholds) +
        {
          fieldConfig+: {
            defaults+: {
              color: { mode: 'palette-classic' },
            },
          },
          options+: {
            reduceOptions+: {
              calcs: ['lastNotNull'],
            },
          },
        };

      local tenantTablePanel(title, description, query, valueName, unit, decimals) =
        $.tablePanel(query, {
          tenant: {
            alias: 'tenant',
            type: 'string',
            unit: 'string',
          },
          Value: {
            alias: valueName,
            decimals: decimals,
            unit: unit,
          },
        }) +
        $.panelDescription(title, description) +
        {
          title: title,
          targets: [
            target { legendFormat: '{{tenant}}' }
            for target in super.targets
          ],
          transformations: [
            {
              id: 'organize',
              options: {
                excludeByName: {},
                includeByName: {
                  Value: true,
                  tenant: true,
                },
                indexByName: {
                  Value: 1,
                  tenant: 0,
                },
                renameByName: {
                  Value: valueName,
                  tenant: 'tenant',
                },
              },
            },
            {
              id: 'sortBy',
              options: {
                fields: {},
                sort: [
                  {
                    desc: true,
                    field: valueName,
                  },
                ],
              },
            },
          ],
        };

      local queryTemplate(name, query, allValue, regex='') = {
        allValue: allValue,
        allowCustomValue: true,
        current: {
          text: 'All',
          value: '$__all',
        },
        datasource: {
          type: 'prometheus',
          uid: '$datasource',
        },
        hide: 0,
        includeAll: true,
        multi: true,
        name: name,
        options: [],
        query: query,
        refresh: 2,
        regex: regex,
        regexApplyTo: 'value',
        skipUrlSync: false,
        sort: 0,
        type: 'query',
      };

      $.dashboard('Tempo / Livestore') {
        description: 'Tempo LiveStore dashboard',
        links: [
          {
            asDropdown: false,
            icon: 'external link',
            includeVars: true,
            keepTime: true,
            tags: [],
            targetBlank: true,
            title: 'Livestore Write Path',
            tooltip: '',
            type: 'link',
            url: '/d/livestore-write-path/livestore-write-path?${__url_time_range}&var-datasource=$datasource&var-namespace=$namespace&var-pod=$pod',
          },
        ],
        refresh: '',
        tags: ['tempo', 'livestore', 'triage'],
        templating+: {
          list+: [
            queryTemplate('namespace', 'label_values(tempo_live_store_records_processed_total, namespace)', 'tempo.*'),
            queryTemplate('cluster', 'label_values(tempo_live_store_records_processed_total{namespace=~"$namespace"}, cluster)', '.*'),
            queryTemplate('pod', 'label_values(kube_pod_container_info{' + namespaceMatcher + ', container="%s", pod=~"%s"}, pod)' % [liveStoreContainer, liveStorePodRegex], liveStorePodRegex, '/^(live-store-zone-[ab]-.*)$/'),
            queryTemplate('tenant', 'label_values(tempo_live_store_traces_created_total{' + namespaceMatcher + '}, tenant)', '.*'),
          ],
        },
      }
      .addRow(
        row('Triage cockpit', [4, 2, 2, 2, 2])
        .addPanel(
          timeseriesPanel(
            'Read latency p99 by zone/route',
            'First read-path signal: p99 live-store read latency split by zone and gRPC route. If this rises, open the Read latency row next.',
            'histogram_quantile(0.99, sum by (le, zone, route) (%s))' % zoneFromPod(metricRate('tempo_request_duration_seconds_bucket', readRequestMatcher)),
            '{{zone}} {{route}}',
            unit='s',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Read QPS by zone',
            'Live-store read request rate by zone. This gives read-traffic context for the p99 latency panel without route-cardinality noise.',
            'sum by (zone) (%s)' % zoneFromPod(metricRate('tempo_request_duration_seconds_count', readRequestMatcher)),
            '{{zone}}',
            unit='ops',
          )
        )
        .addPanel(
          statPanel(
            'Max ingest lag by zone',
            'Worst live-store Kafka lag age by live-store consumer group / zone.',
            'max by (group) (%s)' % metric('tempo_ingest_group_partition_lag_seconds', liveStoreIngestMatcher),
            '{{group}}',
            unit='s',
            decimals=0,
          )
        )
        .addPanel(
          statPanel(
            'Backpressure time by zone',
            'Seconds spent backpressuring per second, split by live-store zone. Uses tempo_live_store_back_pressure_seconds_total when present, with duration histogram sum as fallback for cells that do not expose the counter.',
            'sum by (zone) (%s) or sum by (zone) (%s)' % [
              zoneRate('tempo_live_store_back_pressure_seconds_total'),
              zoneRate('tempo_live_store_back_pressure_duration_seconds_sum'),
            ],
            '{{zone}}',
            unit='percentunit',
            decimals=2,
          )
        )
        .addPanel(
          timeseriesPanel(
            'Complete queue depth by zone',
            'Pending complete-block queue depth by live-store zone. Max highlights the hottest pod; p99 shows broad zone pressure.',
            [
              'max by (zone) (%s)' % zoneMetric('tempo_live_store_complete_queue_length'),
              'quantile by (zone) (0.99, %s)' % zoneMetric('tempo_live_store_complete_queue_length'),
            ],
            ['max {{zone}}', 'p99 {{zone}}'],
          )
        )
      )
      .addRow(
        row('Read latency and query impact', [7, 5])
        .addPanel(
          timeseriesPanel(
            'Query latency p99/p50 by namespace/route',
            'Live-store read latency by cluster, namespace, and gRPC route with live-store zones collapsed by aggregating histogram buckets before quantiles. This is the request-weighted latency view; use zone-specific panels only for diagnostics.',
            [
              'histogram_quantile(0.99, sum by (le, cluster, namespace, route) (%s))' % metricRate('tempo_request_duration_seconds_bucket', readRequestMatcher),
              'histogram_quantile(0.50, sum by (le, cluster, namespace, route) (%s))' % metricRate('tempo_request_duration_seconds_bucket', readRequestMatcher),
            ],
            ['p99 {{cluster}}/{{namespace}} {{route}}', 'p50 {{cluster}}/{{namespace}} {{route}}'],
            unit='s',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Query QPS by namespace/route',
            'Read traffic served by live-store query routes, split by cluster, namespace, and route with live-store zones collapsed.',
            'sum by (cluster, namespace, route) (%s)' % metricRate('tempo_request_duration_seconds_count', readRequestMatcher),
            '{{cluster}}/{{namespace}} {{route}}',
            unit='ops',
          )
        )
      )
      .addRow(
        row('Consistency: lag and throughput', [6, 6, 4, 4, 4])
        .addPanel(
          timeseriesPanel(
            'Ingest lag by zone',
            'Max live-store partition lag age by cluster/namespace and live-store zone.',
            'max by (cluster, namespace, group) (%s)' % metric('tempo_ingest_group_partition_lag_seconds', liveStoreIngestMatcher),
            '{{cluster}}/{{namespace}} {{group}}',
            unit='s',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Worst partition lag by pod',
            'Top lagging partitions. Pod names include the live-store zone; use this to identify skew or stuck consumers.',
            'topk(20, max by (cluster, namespace, pod, partition) (%s))' % metric('tempo_ingest_group_partition_lag_seconds', liveStoreIngestMatcher),
            '{{cluster}}/{{namespace}} {{pod}} p{{partition}}',
            unit='s',
          )
        )
        .addPanel(
          statPanel(
            'Records processed/s by zone',
            'Live-store records processed after Kafka fetch, split by live-store zone. This avoids hiding zone imbalance or double-counting as a single total.',
            'sum by (zone) (%s)' % zoneRate('tempo_live_store_records_processed_total', liveStoreTenantAfterMatcher),
            '{{zone}}',
            unit='ops',
          )
        )
        .addPanel(
          statPanel(
            'Bytes received/s by zone',
            'Consumer-side live-store input bytes/s from the ingest stream, split by live-store zone. This is live-store consumer work, not unique producer-side Kafka ingress.',
            'sum by (zone) (%s)' % zoneRate('tempo_live_store_bytes_received_total', liveStoreTenantAfterMatcher),
            '{{zone}}',
            unit='Bps',
            decimals=0,
          )
        )
        .addPanel(
          timeseriesPanel(
            'Records processed trend by zone',
            'Live-store records successfully processed after Kafka fetch, split by zone. This is not a loss detector; use lag, backpressure, completion queue, readiness, and logs to investigate missing throughput.',
            'sum by (zone) (%s)' % zoneRate('tempo_live_store_records_processed_total'),
            '{{zone}}',
            unit='ops',
          )
        )
      )
      .addRow(
        row('Backpressure and readiness', [6, 6, 6, 6, 6], collapsed=true)
        .addPanel(
          timeseriesPanel(
            'Backpressure time by zone/reason',
            'Backpressure seconds per second by live-store zone and reason. Uses reason when the counter is available; otherwise falls back to duration-sum by zone.',
            'sum by (zone, reason) (%s) or sum by (zone) (%s)' % [
              zoneRate('tempo_live_store_back_pressure_seconds_total'),
              zoneRate('tempo_live_store_back_pressure_duration_seconds_sum'),
            ],
            '{{zone}} {{reason}}',
            unit='percentunit',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Backpressure duration by zone',
            'Duration distribution for individual backpressure stalls by live-store zone.',
            [
              'histogram_quantile(0.50, sum by (le, zone) (%s))' % zoneRate('tempo_live_store_back_pressure_duration_seconds_bucket'),
              'histogram_quantile(0.99, sum by (le, zone) (%s))' % zoneRate('tempo_live_store_back_pressure_duration_seconds_bucket'),
            ],
            ['p50 {{zone}}', 'p99 {{zone}}'],
            unit='s',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Backpressure by pod',
            'Top live-store pods spending time in backpressure. Reason is shown when that counter is available; otherwise duration-sum fallback is shown by pod.',
            'topk(30, sum by (cluster, namespace, pod, reason) (%s)) or topk(30, sum by (cluster, namespace, pod) (%s))' % [
              zoneRate('tempo_live_store_back_pressure_seconds_total'),
              zoneRate('tempo_live_store_back_pressure_duration_seconds_sum'),
            ],
            '{{cluster}}/{{namespace}} {{pod}} {{reason}}',
            unit='percentunit',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Live-store readiness by zone',
            'Ready live-store pods versus pod count, split by zone. Drops indicate lost capacity or pods not serving normally.',
            [
              'sum by (zone) (%s)' % zoneMetric('tempo_live_store_ready'),
              'count by (zone) (%s)' % zoneFromPod('kube_pod_info{' + namespaceMatcher + ', pod=~"$pod"}'),
            ],
            ['ready {{zone}}', 'pods {{zone}}'],
          )
        )
        .addPanel(
          timeseriesPanel(
            'Catch-up duration by pod',
            'Time spent catching up after startup or lag events. Pod names include live-store zone.',
            'topk(20, %s)' % metric('tempo_live_store_catch_up_duration_seconds'),
            '{{cluster}}/{{namespace}} {{pod}}',
            unit='s',
          )
        )
      )
      .addRow(
        row('WAL completion path', [4, 4, 4, 6, 6], collapsed=true)
        .addPanel(
          timeseriesPanel(
            'Completion duration by zone',
            'WAL completion latency by live-store zone. Rising p99 with queue depth indicates completion throughput trouble.',
            [
              'histogram_quantile(0.50, sum by (le, zone) (%s))' % zoneRate('tempo_live_store_completion_duration_seconds_bucket'),
              'histogram_quantile(0.99, sum by (le, zone) (%s))' % zoneRate('tempo_live_store_completion_duration_seconds_bucket'),
            ],
            ['p50 {{zone}}', 'p99 {{zone}}'],
            unit='s',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Blocks completed / failures by zone',
            'Successful block completions, failures, and retries by live-store zone.',
            [
              'sum by (zone) (%s)' % zoneRate('tempo_live_store_blocks_completed_total'),
              'sum by (zone) (%s)' % zoneRate('tempo_live_store_failed_completions_total'),
              'sum by (zone) (%s)' % zoneRate('tempo_live_store_completion_retries_total'),
            ],
            ['completed {{zone}}', 'failed {{zone}}', 'retries {{zone}}'],
            unit='ops',
          )
        )
        .addPanel(
          statPanel(
            'Completion failures/s by zone',
            'WAL complete-block failures split by live-store zone. Non-zero means the completion path needs inspection.',
            'sum by (zone) (%s)' % zoneRate('tempo_live_store_failed_completions_total'),
            '{{zone}}',
            unit='ops',
            decimals=3,
            thresholds=[{ color: 'green', value: null }],
          )
        )
        .addPanel(
          timeseriesPanel(
            'Completion block size by zone',
            'Completed block size distribution by live-store zone. Large shifts can explain completion latency and memory pressure.',
            [
              'histogram_quantile(0.50, sum by (le, zone) (%s))' % zoneRate('tempo_live_store_completion_size_bytes_bucket'),
              'histogram_quantile(0.99, sum by (le, zone) (%s))' % zoneRate('tempo_live_store_completion_size_bytes_bucket'),
            ],
            ['p50 {{zone}}', 'p99 {{zone}}'],
            unit='bytes',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Blocks cut by zone/reason',
            'Rate of WAL block cuts by zone and reason. Use with block size to confirm expected cut behavior.',
            'sum by (zone, reason) (%s)' % zoneRate('tempo_live_store_blocks_cut_total'),
            '{{zone}} {{reason}}',
            unit='ops',
          )
        )
      )
      .addRow(
        row('Live traces and tenant pressure', [6, 6, 6, 6], collapsed=true)
        .addPanel(
          timeseriesPanel(
            'Live traces by zone',
            'Number of in-memory live traces by live-store zone. Sudden growth can create memory pressure and backpressure.',
            'sum by (zone) (%s)' % zoneMetric('tempo_live_store_live_traces', liveStoreTenantMatcher),
            '{{zone}}',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Live trace memory by zone',
            'Bytes retained for live traces by live-store zone.',
            'sum by (zone) (%s)' % zoneMetric('tempo_live_store_live_trace_bytes', liveStoreTenantMatcher),
            '{{zone}}',
            unit='bytes',
          )
        )
        .addPanel(
          tenantTablePanel(
            'Top tenants by traces created/s (current)',
            'Current top tenants by live-store trace creation rate across selected zones. Click a tenant to filter this dashboard or open RT overrides.',
            'topk(10, sum by (tenant) (%s))' % metricRate('tempo_live_store_traces_created_total', liveStoreTenantMatcher),
            'traces/s',
            'ops',
            2,
          )
        )
        .addPanel(
          tenantTablePanel(
            'Top tenants by bytes received (current)',
            'Current top tenants by live-store input byte rate across selected zones. Click a tenant to filter this dashboard or open RT overrides.',
            'topk(10, sum by (tenant) (%s))' % metricRate('tempo_live_store_bytes_received_total', liveStoreTenantMatcher),
            'bytes/s',
            'Bps',
            0,
          )
        )
      )
      .addRow(
        row('Resource pressure', [6, 6, 6, 6], collapsed=true)
        .addPanel(
          timeseriesPanel(
            'CPU pressure by zone',
            'Max and p95 live-store CPU utilization by zone. Use after identifying a skewed zone or queue.',
            [
              'max by (zone) (%s)' % zoneFromPod('container:cpu_utilisation:ratio{' + liveStoreMatcher + '}'),
              'quantile by (zone) (0.95, %s)' % zoneFromPod('container:cpu_utilisation:ratio{' + liveStoreMatcher + '}'),
            ],
            ['max {{zone}}', 'p95 {{zone}}'],
            unit='percentunit',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Memory working set by zone',
            'Max and p95 live-store memory working set by zone.',
            [
              'max by (zone) (%s)' % zoneMetric('container_memory_working_set_bytes'),
              'quantile by (zone) (0.95, %s)' % zoneMetric('container_memory_working_set_bytes'),
            ],
            ['max {{zone}}', 'p95 {{zone}}'],
            unit='bytes',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Go heap allocation by zone',
            'Go heap allocation churn summed by live-store zone. Rising allocation with backpressure suggests GC/memory pressure.',
            'sum by (zone) (%s)' % zoneRate('go_memstats_alloc_bytes_total'),
            '{{zone}}',
            unit='Bps',
          )
        )
        .addPanel(
          timeseriesPanel(
            'Restarts over range by zone',
            'Container restarts for live-store pods during the dashboard time range, summed by zone.',
            'sum by (zone) (%s)' % zoneFromPod('increase(kube_pod_container_status_restarts_total{' + liveStoreMatcher + '}[$__range])'),
            '{{zone}}',
          )
        )
      )
      .addRow(
        row('Autoscaling / KEDA', [3, 3, 3, 3])
        .addPanel(
          timeseriesPanel(
            'KEDA/HPA replicas',
            'Replica bounds and HPA decisions for live-store when present plus the Warpstream agents that feed the live-store path. Some prod cells do not have a live-store HPA.',
            [
              |||
                max by (scaletargetref_name) (
                  kube_horizontalpodautoscaler_spec_min_replicas{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"}
                  * on (cluster, namespace, horizontalpodautoscaler) group_left (scaletargetref_name)
                  group by (cluster, namespace, horizontalpodautoscaler, scaletargetref_name) (kube_horizontalpodautoscaler_info{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"})
                )
              ||| % { namespace_matcher: $.namespaceMatcher(), hpa_regex: liveStoreAndWarpstreamHpaRegex },
              |||
                max by (scaletargetref_name) (
                  kube_horizontalpodautoscaler_status_current_replicas{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"}
                  * on (cluster, namespace, horizontalpodautoscaler) group_left (scaletargetref_name)
                  group by (cluster, namespace, horizontalpodautoscaler, scaletargetref_name) (kube_horizontalpodautoscaler_info{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"})
                )
              ||| % { namespace_matcher: $.namespaceMatcher(), hpa_regex: liveStoreAndWarpstreamHpaRegex },
              |||
                max by (scaletargetref_name) (
                  kube_horizontalpodautoscaler_status_desired_replicas{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"}
                  * on (cluster, namespace, horizontalpodautoscaler) group_left (scaletargetref_name)
                  group by (cluster, namespace, horizontalpodautoscaler, scaletargetref_name) (kube_horizontalpodautoscaler_info{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"})
                )
              ||| % { namespace_matcher: $.namespaceMatcher(), hpa_regex: liveStoreAndWarpstreamHpaRegex },
              |||
                max by (scaletargetref_name) (
                  kube_horizontalpodautoscaler_spec_max_replicas{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"}
                  * on (cluster, namespace, horizontalpodautoscaler) group_left (scaletargetref_name)
                  group by (cluster, namespace, horizontalpodautoscaler, scaletargetref_name) (kube_horizontalpodautoscaler_info{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"})
                )
              ||| % { namespace_matcher: $.namespaceMatcher(), hpa_regex: liveStoreAndWarpstreamHpaRegex },
            ],
            ['min {{scaletargetref_name}}', 'current {{scaletargetref_name}}', 'desired {{scaletargetref_name}}', 'max {{scaletargetref_name}}'],
          )
        )
        .addPanel(
          timeseriesPanel(
            'KEDA CPU desired replicas',
            'KEDA scaler metric divided by the HPA target, following the Warpstream Agent Read dashboard pattern. Shows which Warpstream agent CPU scaler is driving desired replicas.',
            |||
              sum by (scaledObject) (
                max by (cluster, namespace, scaledObject, metric, scaler) (
                  label_replace(keda_scaler_metrics_value{cluster=~"$cluster", exported_namespace=~"$namespace", scaledObject=~"%(scaled_objects)s", scaler=~".*cpu.*|prometheusScaler"}, "namespace", "$1", "exported_namespace", "(.*)")
                )
                / on(cluster, namespace, scaledObject, metric) group_left
                max by (cluster, namespace, scaledObject, metric) (
                  label_replace(label_replace(kube_horizontalpodautoscaler_spec_target_metric{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"}, "metric", "$1", "metric_name", "(.+)"), "scaledObject", "$1", "horizontalpodautoscaler", "keda-hpa-(.*)")
                )
              )
            ||| % { namespace_matcher: $.namespaceMatcher(), scaled_objects: liveStoreAndWarpstreamScaledObjects, hpa_regex: liveStoreAndWarpstreamHpaRegex },
            '{{scaledObject}}',
          )
        )
        .addPanel(
          timeseriesPanel(
            'KEDA memory desired replicas',
            'KEDA memory scaler metric divided by the HPA target, following the Warpstream Agent Read dashboard pattern.',
            |||
              sum by (scaledObject) (
                max by (cluster, namespace, scaledObject, metric, scaler) (
                  label_replace(keda_scaler_metrics_value{cluster=~"$cluster", exported_namespace=~"$namespace", scaledObject=~"%(scaled_objects)s", scaler=~".*memory.*"}, "namespace", "$1", "exported_namespace", "(.*)")
                )
                / on(cluster, namespace, scaledObject, metric) group_left
                max by (cluster, namespace, scaledObject, metric) (
                  label_replace(label_replace(kube_horizontalpodautoscaler_spec_target_metric{%(namespace_matcher)s, horizontalpodautoscaler=~"%(hpa_regex)s"}, "metric", "$1", "metric_name", "(.+)"), "scaledObject", "$1", "horizontalpodautoscaler", "keda-hpa-(.*)")
                )
              )
            ||| % { namespace_matcher: $.namespaceMatcher(), scaled_objects: liveStoreAndWarpstreamScaledObjects, hpa_regex: liveStoreAndWarpstreamHpaRegex },
            '{{scaledObject}}',
          )
        )
        .addPanel(
          timeseriesPanel(
            'KEDA scaler failures',
            'KEDA scaler error rate for live-store when present plus Warpstream agents on the live-store ingest path.',
            |||
              sum by (scaledObject, scaler) (
                label_replace(rate(keda_scaler_errors{cluster=~"$cluster", exported_namespace=~"$namespace", scaledObject=~"%(scaled_objects)s"}[$__rate_interval]), "namespace", "$1", "exported_namespace", "(.+)")
              )
            ||| % { scaled_objects: liveStoreAndWarpstreamScaledObjects },
            '{{scaledObject}} {{scaler}}',
          )
        )
      ) {
        description: 'Tempo LiveStore dashboard',
        refresh: '',
        tags: ['tempo', 'livestore', 'triage'],
      },
  },
}
