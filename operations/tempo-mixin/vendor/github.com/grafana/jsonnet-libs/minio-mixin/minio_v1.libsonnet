local g = import 'grafana-builder/grafana.libsonnet';

local panel_settings_bytes = {
  yaxes: g.yaxes('bytes'),
};

local panel_settings_qps = {
  stack: true,
};

// Add common filters
local f(s) = s % 'instance=~"$instance", job=~"$job"';

g.dashboard('MinIO distributed cluster metrics', std.md5('minio_v1'))
.addTemplate('job', 'minio_version_info', 'job')
.addMultiTemplate('instance', 'minio_version_info{job="$job"}', 'instance')
.addMultiTemplate('disk', 'disk_storage_available{job="$job"}', 'disk')
.addRow(
  g.row('Overview')
  .addPanel(
    g.panel('Storage Used') +
    g.statPanel('sum(disk_storage_used{disk=~"$disk", job=~"$job"}) by (disk) / sum(disk_storage_total{disk=~"$disk", job=~"$job"}) by (disk)') +
    {
      type: 'gauge',
      targets: [super.targets[0] {
        legendFormat: '{{disk}}',
      }],
      // The default 'percentunit' format of statPanel() doesn't seem to have an effect
      fieldConfig: {
        defaults: {
          unit: 'percentunit',
        },
      },
    }
  )
  .addPanel(
    g.panel('Backend Disks') + {
      description: 'Total number of disks in Erasure-type backends. This metric is not available for FileSystem backends.',
    } +
    g.statPanel(f('sum(minio_disks_total{%s}) > 0'), 'none')
  )
  .addPanel(
    g.panel('Backend Disks Offline') +
    g.statPanel(f('sum(minio_disks_offline{%s})'), 'none')
  )
  .addPanel(
    g.panel('Errors') +
    g.queryPanel(f('s3_errors_total{%s}'), '')
  )
)
.addRow(
  g.row('Storage')
  .addPanel(
    g.panel('Storage Used') +
    g.queryPanel(f('disk_storage_used{disk=~"$disk",%s}'), '') +
    panel_settings_bytes,
  )
  .addPanel(
    g.panel('Storage Available') +
    g.queryPanel(f('disk_storage_available{disk=~"$disk",%s}'), '') +
    panel_settings_bytes,
  )
  .addPanel(
    g.panel('Storage Total') +
    g.queryPanel(f('disk_storage_total{disk=~"$disk",%s}'), '') +
    panel_settings_bytes,
  )
)
.addRow(
  g.row('Buckets')
  .addPanel(
    g.panel('Total Size') +
    // Every node reports the same stats, therefore reduce with max
    g.queryPanel(f('max(bucket_usage_size{%s}) by (bucket)'), '{{bucket}}') +
    panel_settings_bytes,
  )
  .addPanel(
    g.panel('Object Count') +
    // Every node reports the same stats, therefore reduce with max
    g.queryPanel(f('max(bucket_objects_count{%s}) by (bucket)'), '{{bucket}}')
  )
  .addPanel(
    g.panel('Object Counts Per Size') +
    // Every node reports the same stats, therefore reduce with max
    g.queryPanel(f('max(bucket_objects_histogram{%s}) by (bucket, object_size)'), '{{bucket}}-{{object_size}}')
  )
)
.addRow(
  g.row('Requests')
  .addPanel(
    g.panel('Read rps') +
    g.queryPanel(f('sum(rate(s3_requests_total{api=~"get.*|list.*|head.*",%s}[$__rate_interval])) by (api)'), '{{api}}') +
    panel_settings_qps
  )
  .addPanel(
    g.panel('Write rps') +
    g.queryPanel(f('sum(rate(s3_requests_total{api=~"put.*",%s}[$__rate_interval])) by (api)'), '{{api}}') +
    panel_settings_qps
  )
  .addPanel(
    g.panel('Delete rps') +
    g.queryPanel(f('sum(rate(s3_requests_total{api=~"delete.*",%s}[$__rate_interval])) by (api)'), '{{api}}')
  )
)
.addRow(
  g.row('Performance')
  .addPanel(
    g.panel('Read latency') +
    g.latencyPanel('s3_ttfb_seconds', f('{api=~"get.*|list.*|head.*",%s}'))
  )
  .addPanel(
    g.panel('Internode traffic') + {
      description: 'Internode traffic for multi-node clusters. Will be zero for single node configurations.',
    } +
    g.queryPanel([
      f('rate(internode_rx_bytes_total{%s}[$__rate_interval])'),
      f('rate(internode_tx_bytes_total{%s}[$__rate_interval])'),
    ], [
      'Inbound-{{instance}}',
      'Outbound-{{instance}}',
    ],)
  )
)
