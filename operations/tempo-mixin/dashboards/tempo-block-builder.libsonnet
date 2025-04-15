local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';

dashboard_utils {
  grafanaDashboards+: {
    'tempo-block-builder.json':
      $.dashboard('Tempo / Block builder')
      .addClusterSelectorTemplates()
      .addRow(
        $.row('Fetched records')
        .addPanel(
          $.panel('Kafka fetched records / sec') +
          $.panelDescription(
            'Kafka fetched records / sec',
            'Overview of per-second rate of records fetched from Kafka.',
          ) +
          $.queryPanel(
            [
              'sum (rate(tempo_block_builder_fetch_records_total{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) },
              'sum (rate(tempo_block_builder_fetch_errors_total{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) },
            ],
            ['sucessful', 'read errors']
          ) +
          $.stack,
        )
        .addPanel(
          $.timeseriesPanel('Per pod Kafka fetched records / sec') +
          $.panelDescription(
            'Per pod Kafka fetched records / sec',
            'Overview of per-second rate of records fetched from Kafka split by pods.',
          ) +
          $.queryPanel(
            'sum by (pod) (rate(tempo_block_builder_fetch_records_total{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) },
            '{{pod}}'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'short' } } },
        )
        .addPanel(
          $.timeseriesPanel('Per partition Kafka fetched records / sec') +
          $.panelDescription(
            'Per partition Kafka fetched records / sec',
            'Overview of per-second rate of records fetched from Kafka split by partition.',
          ) +
          $.queryPanel(
            'sum by (partition) (rate(tempo_block_builder_fetch_records_total{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) },
            '{{partition}}'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'short' } } },
        )
      )
      .addRow(
        $.row('Read bytes')
        .addPanel(
          $.panel('Kafka read bytes / sec') +
          $.panelDescription(
            'Kafka read bytes / sec',
            'Overview of per-second rate of bytes readed from Kafka.',
          ) +
          $.queryPanel(
            'sum (rate(tempo_block_builder_fetch_bytes_total{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) }, 'readed'

          ) {
            yaxes: $.yaxes('binBps'),
          } +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'binBps' } } },
        )
        .addPanel(
          $.timeseriesPanel('Per pod Kafka read bytes / sec') +
          $.panelDescription(
            'Per pod Kafka read bytes / sec',
            'Overview of per-second rate of bytes readed from Kafka split by pods.',
          ) +
          $.queryPanel(
            'sum by (pod) (rate(tempo_block_builder_fetch_bytes_total{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) },
            '{{pod}}'
          ) {
            yaxes: $.yaxes('binBps'),
          } +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'binBps' } } },
        )
        .addPanel(
          $.timeseriesPanel('Per partition Kafka read bytes / sec') +
          $.panelDescription(
            'Per partition Kafka read bytes / sec',
            'Overview of per-second rate of bytes readed from Kafka split by partition.',
          ) +
          $.queryPanel(
            'sum by (partition) (rate(tempo_block_builder_fetch_bytes_total{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) },
            '{{partition}}'
          ) {
            yaxes: $.yaxes('binBps'),
          } +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'binBps' } } },
        )
      )
      .addRow(
        $.row('Flushed blocks')
        .addPanel(
          $.panel('Flushed blocks / sec') +
          $.panelDescription(
            'Block builder partition section duration',
            'Overview of the partition section duration.',
          ) +
          $.queryPanel(
            'sum (rate(tempo_block_builder_flushed_blocks{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) }, 'blocks'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'short' } } },
        )
        .addPanel(
          $.timeseriesPanel('Per pod flushed blocks / sec') +
          $.panelDescription(
            'Block builder partition section duration',
            'Overview of the partition section duration.',
          ) +
          $.queryPanel(
            'sum by (pod) (rate(tempo_block_builder_flushed_blocks{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) },
            '{{pod}}'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'short' } } },
        )
      )
      .addRow(
        $.row('Partitions')
        .addPanel(
          $.timeseriesPanel('Lag of records by partition') +
          $.panelDescription(
            'Kafka lag by partition records',
            'Overview of the lag by partition in records.',
          ) +
          $.queryPanel(
            'max(tempo_ingest_group_partition_lag{%(job)s}) by (partition)' % { job: $.jobMatcher($._config.jobs.block_builder) }, '{{partition}}'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'short' } } },
        )
        .addPanel(
          $.timeseriesPanel('Lag by partition (sec)') +
          $.panelDescription(
            'Kafka lag by partition in seconds',
            'Overview of the lag by partition in seconds.',
          ) +
          $.queryPanel(
            'max(tempo_ingest_group_partition_lag_seconds{%(job)s}) by (partition)' % { job: $.jobMatcher($._config.jobs.block_builder) }, '{{partition}}'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 's' } } },
        )
      )
      .addRow(
        $.row('Consumption time')
        .addPanel(
          $.panel('Partition section duration (sec)') {
            type: 'heatmap',
          } +
          $.panelDescription(
            'Block builder partition section duration',
            'Overview of the partition section duration.',
          ) +
          $.queryPanel(
            'sum(rate(tempo_block_builder_process_partition_section_duration_seconds{%(job)s}[$__rate_interval]))  by (partition)' % { job: $.jobMatcher($._config.jobs.block_builder) }, '{{partition}}'
          ) {
            yaxes: $.yaxes('s'),
          } +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 's' } } },
        )
        .addPanel(
          $.panel('Partition cycle duration (sec)') {
            type: 'heatmap',
          } +
          $.panelDescription(
            'Block builder partition cycle duration',
            'Overview of the partition cycle duration.',
          ) +
          $.queryPanel(
            'sum(rate(tempo_block_builder_consume_cycle_duration_seconds{%(job)s}[$__rate_interval]))' % { job: $.jobMatcher($._config.jobs.block_builder) }, 'cycle duration'
          ) {
            yaxes: $.yaxes('s'),
          } +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 's' } } },
        )
      )

      .addRow(
        $.row('Resources')
        .addPanel(
          $.containerCPUUsagePanel('CPU', $._config.jobs.block_builder),
        )
        .addPanel(
          $.containerMemoryWorkingSetPanel('Memory (workingset)', $._config.jobs.block_builder),
        )
      ),
  },
}
