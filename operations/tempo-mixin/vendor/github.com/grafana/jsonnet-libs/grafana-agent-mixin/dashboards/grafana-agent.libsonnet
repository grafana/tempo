local utils = import '../lib/utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';
local grafana = import 'grafonnet/grafana.libsonnet';

local dashboard = grafana.dashboard;
local row = grafana.row;
local singlestat = grafana.singlestat;
local prometheus = grafana.prometheus;
local graphPanel = grafana.graphPanel;
local tablePanel = grafana.tablePanel;
local template = grafana.template;
local timeSeries = grafana.timeSeries;

local host_matcher = 'job=~"$job", instance=~"$instance"';

// Templates
local ds_template = {
  current: {
    text: 'default',
    value: 'default',
  },
  hide: 0,
  label: 'Data Source',
  name: 'prometheus_datasource',
  options: [],
  query: 'prometheus',
  refresh: 1,
  regex: '',
  type: 'datasource',
};

local job_template = grafana.template.new(
  'job',
  '$prometheus_datasource',
  'label_values(agent_build_info, job)',
  label='job',
  refresh='load',
  multi=true,
  includeAll=true,
  sort=1,
);

local instance_template = grafana.template.new(
  'instance',
  '$prometheus_datasource',
  'label_values(agent_build_info{job=~"$job"}, instance)',
  label='instance',
  refresh='load',
  multi=true,
  includeAll=true,
  sort=1,
);

{
  grafanaDashboards+:: {
    'grafana-agent-overview.json':
      local agentStats =
        tablePanel.new(
          'Running Instances',
          description='General statistics of running grafana agent instances.',
          datasource='$prometheus_datasource',
          span=12,
          styles=[
            { alias: 'Container', pattern: 'container' },
            { alias: 'Instance', pattern: 'instance' },
            { alias: 'Pod', pattern: 'pod' },
            { alias: 'Version', pattern: 'version' },
            { alias: 'Uptime', pattern: 'Value #B', type: 'number', unit: 's' },
          ],
        )
        .addTarget(prometheus.target(
          'count by (instance, version) (agent_build_info{' + host_matcher + '})',
          format='table',
          instant=true,
        ))
        .addTarget(prometheus.target(
          'max by (instance) (time() - process_start_time_seconds{' + host_matcher + '})',
          format='table',
          instant=true,
        ))
        .hideColumn('Time')
        .hideColumn('Value #A');

      local prometheusTargetSync =
        graphPanel.new(
          'Target Sync',
          description='Actual interval to sync the scrape pool.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'sum(rate(prometheus_target_sync_length_seconds_sum{' + host_matcher + '}[$__rate_interval])) by (instance, scrape_job)',
          legendFormat='{{instance}}/{{scrape_job}}',
        )) +
        utils.timeSeriesOverride(unit='s');

      local prometheusTargets =
        graphPanel.new(
          'Targets',
          description='Discovered targets by prometheus service discovery.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'sum by (instance) (prometheus_sd_discovered_targets{' + host_matcher + '})',
        )) +
        utils.timeSeriesOverride(unit='short');

      local averageScrapeIntervalDuration =
        graphPanel.new(
          'Average Scrape Interval Duration',
          description='Actual intervals between scrapes.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(prometheus_target_interval_length_seconds_sum{' + host_matcher + '}[$__rate_interval]) / rate(prometheus_target_interval_length_seconds_count{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}} {{interval}} configured',
        )) +
        utils.timeSeriesOverride(unit='s');

      local scrapeFailures =
        graphPanel.new(
          'Scrape failures',
          description='Shows all scrape failures (sample limit exceeded, duplicate, out of bounds, out of order).',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'sum by (job) (rate(prometheus_target_scrapes_exceeded_sample_limit_total{' + host_matcher + '}[$__rate_interval]))',
          legendFormat='exceeded sample limit: {{job}}'
        ))
        .addTarget(prometheus.target(
          'sum by (job) (rate(prometheus_target_scrapes_sample_duplicate_timestamp_total{' + host_matcher + '}[$__rate_interval]))',
          legendFormat='duplicate timestamp: {{job}}'
        ))
        .addTarget(prometheus.target(
          'sum by (job) (rate(prometheus_target_scrapes_sample_out_of_bounds_total{' + host_matcher + '}[$__rate_interval]))',
          legendFormat='out of bounds: {{job}}'
        ))
        .addTarget(prometheus.target(
          'sum by (job) (rate(prometheus_target_scrapes_sample_out_of_order_total{' + host_matcher + '}[$__rate_interval]))',
          legendFormat='out of order: {{job}}'
        )) +
        utils.timeSeriesOverride(unit='short');

      local appendedSamples =
        graphPanel.new(
          'Appended Samples',
          description='Total number of samples appended to the WAL.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'sum by (job, instance_group_name) (rate(agent_wal_samples_appended_total{' + host_matcher + '}[$__rate_interval]))',
          legendFormat='{{job}} {{instance_group_name}}',
        )) +
        utils.timeSeriesOverride(unit='short');


      grafana.dashboard.new('Grafana Agent Overview', tags=$._config.dashboardTags, editable=false, time_from='%s' % $._config.dashboardPeriod, uid='integration-agent')
      .addTemplates([
        ds_template,
        job_template,
        instance_template,
      ])
      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Grafana Agent Dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      ))
      .addRow(
        row.new('Overview')
        .addPanel(agentStats)
      )
      .addRow(
        row.new('Prometheus Discovery')
        .addPanel(prometheusTargetSync)
        .addPanel(prometheusTargets)
      )
      .addRow(
        row.new('Prometheus Retrieval')
        .addPanel(averageScrapeIntervalDuration)
        .addPanel(scrapeFailures)
        .addPanel(appendedSamples)
      ),

    // Remote write specific dashboard.
    'grafana-agent-remote-write.json':
      local timestampComparison =
        graphPanel.new(
          'Highest Timestamp In vs. Highest Timestamp Sent',
          description='Highest timestamp that has come into the remote storage via the Appender interface, in seconds since epoch.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          '\n            (\n              prometheus_remote_storage_highest_timestamp_in_seconds{' + host_matcher + '}\n              -\n              ignoring(url, remote_name) group_right(pod)\n              prometheus_remote_storage_queue_highest_sent_timestamp_seconds{' + host_matcher + '}\n            )\n          ',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='s');

      local remoteSendLatency =
        graphPanel.new(
          'Latency Over Rate Interval',
          description='Rate of duration of send calls to the remote storage',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(prometheus_remote_storage_sent_batch_duration_seconds_sum{' + host_matcher + '}[$__rate_interval]) / rate(prometheus_remote_storage_sent_batch_duration_seconds_count{' + host_matcher + '}[$__rate_interval])',
          legendFormat='mean {{instance}}',
        ))
        .addTarget(prometheus.target(
          'histogram_quantile(0.99, rate(prometheus_remote_storage_sent_batch_duration_seconds_bucket{' + host_matcher + '}[$__rate_interval]))',
          legendFormat='p99 {{instance}}',
        )) +
        utils.timeSeriesOverride(unit='s');

      local samplesInRate =
        graphPanel.new(
          'Samples In Rate Over Rate Interval',
          description='Rate of total number of samples appended to the WAL.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(agent_wal_samples_appended_total{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local samplesOutRate =
        graphPanel.new(
          'Samples Out Rate Over Rate Interval',
          description='Rate of total number of samples sent to remote storage.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(prometheus_remote_storage_succeeded_samples_total{' + host_matcher + '}[$__rate_interval]) or rate(prometheus_remote_storage_samples_total{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local currentShards =
        graphPanel.new(
          'Current Shards',
          description='The number of shards used for parallel sending to the remote storage.',
          datasource='$prometheus_datasource',
          span=12,
          min_span=6,
        )
        .addTarget(prometheus.target(
          'prometheus_remote_storage_shards{' + host_matcher + '}',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local maxShards =
        graphPanel.new(
          'Max Shards',
          description='The maximum number of shards that the queue is allowed to run.',
          datasource='$prometheus_datasource',
          span=4,
        )
        .addTarget(prometheus.target(
          'prometheus_remote_storage_shards_max{' + host_matcher + '}',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local minShards =
        graphPanel.new(
          'Min Shards',
          description='The minimum number of shards that the queue is allowed to run.',
          datasource='$prometheus_datasource',
          span=4,
        )
        .addTarget(prometheus.target(
          'prometheus_remote_storage_shards_min{' + host_matcher + '}',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local desiredShards =
        graphPanel.new(
          'Desired Shards',
          description='The number of shards that the queues shard calculation wants to run based on the rate of samples in vs. samples out.',
          datasource='$prometheus_datasource',
          span=4,
        )
        .addTarget(prometheus.target(
          'prometheus_remote_storage_shards_desired{' + host_matcher + '}',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local shardsCapacity =
        graphPanel.new(
          'Shard Capacity',
          description='The capacity of each shard of the queue used for parallel sending to the remote storage.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'prometheus_remote_storage_shard_capacity{' + host_matcher + '}',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local pendingSamples =
        graphPanel.new(
          'Pending Samples',
          description='The number of samples pending in the queues shards to be sent to the remote storage.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'prometheus_remote_storage_samples_pending{' + host_matcher + '}',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local queueSegment =
        graphPanel.new(
          'Remote Write Current Segment',
          description='Current segment the WAL watcher is reading records from.',
          datasource='$prometheus_datasource',
          span=6,
          formatY1='none',
        )
        .addTarget(prometheus.target(
          'prometheus_wal_watcher_current_segment{' + host_matcher + '}',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local droppedSamples =
        graphPanel.new(
          'Dropped Samples',
          description='Total number of samples which were dropped after being read from the WAL before being sent via remote write, either via relabelling or unintentionally because of an unknown reference ID.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(prometheus_remote_storage_samples_dropped_total{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local failedSamples =
        graphPanel.new(
          'Failed Samples',
          description='Total number of samples which failed on send to remote storage, non-recoverable errors.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(prometheus_remote_storage_samples_failed_total{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local retriedSamples =
        graphPanel.new(
          'Retried Samples',
          description='Total number of samples which failed on send to remote storage but were retried because the send error was recoverable.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(prometheus_remote_storage_samples_retried_total{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local enqueueRetries =
        graphPanel.new(
          'Enqueue Retries',
          description='Total number of times enqueue has failed because a shards queue was full.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(prometheus_remote_storage_enqueue_retries_total{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='short');

      grafana.dashboard.new('Grafana Agent Remote Write', tags=$._config.dashboardTags, editable=false, time_from='%s' % $._config.dashboardPeriod, uid='integration-agent-prom-rw')
      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Grafana Agent Dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      ))
      .addTemplates([
        ds_template,
        job_template,
        instance_template,
      ])
      .addRow(
        row.new('Timestamps')
        .addPanel(timestampComparison)
        .addPanel(remoteSendLatency)
      )
      .addRow(
        row.new('Samples')
        .addPanel(samplesInRate)
        .addPanel(samplesOutRate)
        .addPanel(pendingSamples)
        .addPanel(droppedSamples)
        .addPanel(failedSamples)
        .addPanel(retriedSamples)
      )
      .addRow(
        row.new('Shards')
        .addPanel(currentShards)
        .addPanel(maxShards)
        .addPanel(minShards)
        .addPanel(desiredShards)
      )
      .addRow(
        row.new('Shard Details')
        .addPanel(shardsCapacity)
      )
      .addRow(
        row.new('Segments')
        .addPanel(queueSegment)
      )
      .addRow(
        row.new('Misc. Rates')
        .addPanel(enqueueRetries)
      ),

    'grafana-agent-tracing-pipeline.json':
      local acceptedSpans =
        graphPanel.new(
          'Accepted spans',
          description='Number of spans successfully pushed into the pipeline.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=3,
          legend_show=false,
          fill=0,
        )
        .addTarget(prometheus.target(
          'rate(traces_receiver_accepted_spans{' + host_matcher + ',receiver!="otlp/lb"}[$__rate_interval])',
          legendFormat='{{ instance }} - {{ receiver }}/{{ transport }}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local refusedSpans =
        graphPanel.new(
          'Refused spans',
          description='Number of spans that could not be pushed into the pipeline.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=3,
          legend_show=false,
          fill=0,
        )
        .addTarget(prometheus.target(
          'rate(traces_receiver_refused_spans{' + host_matcher + ',receiver!="otlp/lb"}[$__rate_interval])',
          legendFormat='{{ instance }} - {{ receiver }}/{{ transport }}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local sentSpans =
        graphPanel.new(
          'Exported spans',
          description='Number of spans successfully sent to destination.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=3,
          legend_show=false,
          fill=0,
        )
        .addTarget(prometheus.target(
          'rate(traces_exporter_sent_spans{' + host_matcher + ',exporter!="otlp"}[$__rate_interval])',
          legendFormat='{{ instance }} - {{ exporter }}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local exportedFailedSpans =
        graphPanel.new(
          'Exported failed spans',
          description='Number of spans failed to send.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=3,
          legend_show=false,
          fill=0,
        )
        .addTarget(prometheus.target(
          'rate(traces_exporter_send_failed_spans{' + host_matcher + ',exporter!="otlp"}[$__rate_interval])',
          legendFormat='{{ instance }} - {{ exporter }}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local receivedSpans(receiverFilter, width) =
        graphPanel.new(
          'Received spans',
          description='Number of spans successfully pushed into the pipeline.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=width,
          fill=1,
        )
        .addTarget(prometheus.target(
          'sum(rate(traces_receiver_accepted_spans{' + host_matcher + ',%s}[$__rate_interval]))' % receiverFilter,
          legendFormat='Accepted',
        ))
        .addTarget(prometheus.target(
          'sum(rate(traces_receiver_refused_spans{' + host_matcher + ',%s}[$__rate_interval])) ' % receiverFilter,
          legendFormat='Refused',
        )) +
        utils.timeSeriesOverride(unit='short');

      local exportedSpans(exporterFilter, width) =
        graphPanel.new(
          'Exported spans',
          description='Number of spans successfully sent to destination.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=width,
          fill=1,
        )
        .addTarget(prometheus.target(
          'sum(rate(traces_exporter_sent_spans{' + host_matcher + ',%s}[$__rate_interval]))' % exporterFilter,
          legendFormat='Sent',
        ))
        .addTarget(prometheus.target(
          'sum(rate(traces_exporter_send_failed_spans{' + host_matcher + ',%s}[$__rate_interval]))' % exporterFilter,
          legendFormat='Send failed',
        )) +
        utils.timeSeriesOverride(unit='short');

      local loadBalancedSpans =
        graphPanel.new(
          'Load-balanced spans',
          description='Number of load balanced spans.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=3,
          fill=1,
          stack=true,
        )
        .addTarget(prometheus.target(
          'rate(traces_loadbalancer_backend_outcome{' + host_matcher + ', cluster=~"$cluster",namespace=~"$namespace",success="true",container=~"$container",pod=~"$pod"}[$__rate_interval])',
          legendFormat='{{ pod }}',
        )) +
        utils.timeSeriesOverride(unit='short');

      local peersNum =
        graphPanel.new(
          'Number of peers',
          description='Number of tracing peers.',
          datasource='$prometheus_datasource',
          interval='1m',
          span=3,
          legend_show=false,
          fill=0,
        )
        .addTarget(prometheus.target(
          'traces_loadbalancer_num_backends{' + host_matcher + '}',
          legendFormat='{{ pod }}',
        )) +
        utils.timeSeriesOverride(unit='short');

      dashboard.new('Grafana Agent Tracing Pipeline', tags=$._config.dashboardTags, editable=false, time_from='%s' % $._config.dashboardPeriod, uid='integration-agent-tracing-pl')
      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Grafana Agent Dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      ))
      .addTemplates([
        ds_template,
        job_template,
        instance_template,
      ])
      .addRow(
        row.new('Write / Read')
        .addPanel(acceptedSpans)
        .addPanel(refusedSpans)
        .addPanel(sentSpans)
        .addPanel(exportedFailedSpans)
        .addPanel(receivedSpans('receiver!="otlp/lb"', 6))
        .addPanel(exportedSpans('exporter!="otlp"', 6))
      )
      .addRow(
        row.new('Load balancing')
        .addPanel(loadBalancedSpans)
        .addPanel(peersNum)
        .addPanel(receivedSpans('receiver="otlp/lb"', 3))
        .addPanel(exportedSpans('exporter="otlp"', 3))
      ),
  },
}
