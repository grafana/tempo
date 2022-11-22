local grafana = import 'github.com/grafana/grafonnet-lib/grafonnet/grafana.libsonnet';
local prometheus = grafana.prometheus;
local nsqgrafonnet = import '../lib/nsqgrafonnet/nsqgrafonnet.libsonnet';

{

  grafanaDashboards+:: {

    local nsqSelector = 'job=~"$job", instance=~"$instance"',
    local nsqTopicSelector = nsqSelector + ',topic=~"$topic"',
    local nsqChannelSelector = nsqTopicSelector + ',channel=~"$channel"',
    local nsqTopicDataLinkQueryParams = '${datasource:queryparam}&${job:queryparam}&${__url_time_range}',
    local nsqChannelDataLinkQueryParams = '${datasource:queryparam}&${job:queryparam}&${__url_time_range}',
    local nsqHeapMemory =
      nsqgrafonnet.timeseries.new(
        'Heap memory (avg)',
        description='Average memory usage if instance=All filter is selected'
      )
      .addTarget(prometheus.target(expr='avg by (job) (nsq_mem_heap_in_use_bytes{%s})' % nsqSelector, intervalFactor=2, legendFormat='heap in use'))
      .addTarget(prometheus.target(expr='avg by (job) (nsq_mem_heap_idle_bytes{%s})' % nsqSelector, intervalFactor=2, legendFormat='heap idle bytes'))
      .addTarget(prometheus.target(expr='avg by (job) (nsq_mem_heap_released_bytes{%s})' % nsqSelector, intervalFactor=2, legendFormat='heap bytes released'))
      .withFillOpacity(5)
      .withUnit('decbytes'),

    local nsqHeapObjects =
      nsqgrafonnet.timeseries.new('Heap objects')
      .addTarget(prometheus.target(expr='nsq_mem_heap_objects{%s}' % nsqSelector, intervalFactor=2, legendFormat='{{ instance }}'))
      .withFillOpacity(5)
      .withUnit('short'),

    local nsqNextGC =
      nsqgrafonnet.timeseries.new(
        'Next GC in bytes',
        description='The number used bytes at which the runtime plans to perform the next garbage collection'
      )
      .addTarget(prometheus.target(expr='nsq_mem_next_gc_bytes{%s}' % nsqSelector, intervalFactor=2, legendFormat='{{ instance }}'))
      .withFillOpacity(5)
      .withUnit('decbytes'),

    local nsqGCpause =
      nsqgrafonnet.timeseries.new('GC pause time')
      .addTarget(prometheus.target(expr='nsq_mem_gc_pause_usec_100{%s}' % nsqSelector, intervalFactor=2, legendFormat='p100 {{ instance }}', hide=true))
      .addTarget(prometheus.target(expr='nsq_mem_gc_pause_usec_99{%s}' % nsqSelector, intervalFactor=2, legendFormat='p99 {{ instance }}', hide=false))
      .addTarget(prometheus.target(expr='nsq_mem_gc_pause_usec_95{%s}' % nsqSelector, intervalFactor=2, legendFormat='p95 {{ instance }}', hide=true))
      .withUnit('µs')
      .withFillOpacity(5),

    local nsqTopics =
      nsqgrafonnet.timeseries.new('$topic topic messages')
      .addTarget(prometheus.target(expr='avg by (instance) (rate(nsq_topic_message_count{%s}[$__rate_interval]))' % nsqTopicSelector, intervalFactor=1, legendFormat='{{ instance }} rps'))
      .withUnit('reqps')
      .addDataLink(
        title='Show split by topics for ${__field.labels.instance}',
        url='d/nsq-topics?var-instance=${__field.labels.instance}&%s' % nsqTopicDataLinkQueryParams
      ),

    local nsqTopicsDepth =
      nsqgrafonnet.timeseries.new('$topic topic depth')
      .addTarget(prometheus.target(expr='sum by (instance) (nsq_topic_depth{%s})' % nsqTopicSelector, intervalFactor=1, legendFormat='{{ instance }} depth'))
      .addTarget(prometheus.target(expr='sum by (instance) (nsq_topic_backend_depth{%s})' % nsqTopicSelector, intervalFactor=1, legendFormat='{{ instance }} memory+disk depth'))
      .withUnit('short')
      .addDataLink(
        title='Show split by topics for ${__field.labels.instance}',
        url='d/nsq-topics?var-instance=${__field.labels.instance}&%s' % nsqTopicDataLinkQueryParams
      ),

    local nsqChannelClients =
      nsqgrafonnet.timeseries.new('$channel channel clients')
      .addTarget(prometheus.target(expr='sum by (instance) (nsq_topic_channel_clients{%s})' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }}'))
      .addDataLink(
        title='Show split by channels for ${__field.labels.instance}',
        url='d/nsq-topics?var-instance=${__field.labels.instance}&%s' % nsqChannelDataLinkQueryParams
      ),

    local nsqChannelStats =
      nsqgrafonnet.timeseries.new('$channel channel stats')
      .addTarget(prometheus.target(expr='sum by (instance) (nsq_topic_channel_depth{%s})' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }} depth'))
      .addTarget(prometheus.target(expr='sum by (instance) (nsq_topic_channel_backend_depth{%s})' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }} memory+disk depth'))
      .addTarget(prometheus.target(expr='sum by (instance) (nsq_topic_channel_in_flight_count{%s})' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }} in-flight'))
      .addTarget(prometheus.target(expr='avg by (instance) (rate(nsq_topic_channel_requeue_count{%s}[$__rate_interval]))' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }} requeue'))
      .addTarget(prometheus.target(expr='avg by (instance) (rate(nsq_topic_channel_timeout_count{%s}[$__rate_interval]))' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }} timeout'))
      .addTarget(prometheus.target(expr='sum by (instance) (nsq_topic_channel_deferred_count{%s})' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }} deferred'))
      .withUnit('short')
      .addDataLink(
        title='Show split by channels for ${__field.labels.instance}',
        url='d/nsq-topics?var-instance=${__field.labels.instance}&%s' % nsqChannelDataLinkQueryParams
      ),

    local nsqTopicsE2e =
      nsqgrafonnet.timeseries.new('$topic topic end-to-end processing time')
      .addTarget(prometheus.target(expr='avg by (instance) (nsq_topic_e2e_processing_latency_99{%s})' % nsqTopicSelector, intervalFactor=1, legendFormat='{{ instance }} p99'))
      .addDataLink(
        title='Show split by topics for ${__field.labels.instance}',
        url='d/nsq-topics?var-instance=${__field.labels.instance}&%s' % nsqTopicDataLinkQueryParams
      )
      .withUnit('ns'),

    local nsqChannelE2e =
      nsqgrafonnet.timeseries.new('$channel channel end-to-end processing time')
      .addTarget(prometheus.target(expr='avg by (instance) (nsq_topic_channel_e2e_processing_latency_99{%s})' % nsqChannelSelector, intervalFactor=1, legendFormat='{{ instance }} p99'))
      .addDataLink(
        title='Show split by channels for ${__field.labels.instance}',
        url='d/nsq-topics?var-instance=${__field.labels.instance}&%s' % nsqChannelDataLinkQueryParams
      )
      .withUnit('ns'),

    'nsq-instances.json':
      grafana.dashboard.new(
        '%s Instances' % $._config.dashboardNamePrefix,
        time_from='%s' % $._config.dashboardPeriod,
        editable=false,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        graphTooltip='shared_crosshair',
        uid='nsq-instances'
      )
      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Other NSQ dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      ))
      .addTemplate(
        {
          current: {
            text: 'Prometheus',
            value: 'Prometheus',
          },
          hide: 0,
          label: 'Data Source',
          name: 'datasource',
          options: [],
          query: 'prometheus',
          refresh: 1,
          regex: '',
          type: 'datasource',
        }
      )
      .addTemplate(
        {
          hide: 0,
          label: 'job',
          name: 'job',
          includeAll: true,
          allValue: '.+',
          multi: true,
          options: [],
          query: 'label_values(nsq_topic_message_count, job)',
          refresh: 1,
          regex: '',
          type: 'query',
          datasource: '$datasource',
        },
      )
      .addTemplate(
        {
          hide: 0,
          label: null,
          name: 'instance',
          includeAll: true,
          multi: true,
          options: [],
          query: 'label_values(nsq_topic_message_count{job=~"$job"},instance)',
          refresh: 2,
          regex: '',
          type: 'query',
          datasource: '$datasource',
        },
      )
      .addTemplate(
        {
          hide: 0,
          label: null,
          name: 'topic',
          includeAll: true,
          multi: true,
          options: [],
          query: 'label_values(nsq_topic_message_count{job=~"$job",instance=~"$instance"},topic)',
          refresh: 2,
          regex: '',
          type: 'query',
          datasource: '$datasource',
        },
      )
      .addTemplate(
        {
          hide: 0,
          label: null,
          name: 'channel',
          includeAll: true,
          multi: true,
          options: [],
          query: 'label_values(nsq_topic_channel_message_count{job=~"$job",instance=~"$instance",topic=~"$topic"},channel)',
          refresh: 2,
          regex: '',
          type: 'query',
          datasource: '$datasource',
        },
      )
      .addPanel(grafana.row.new(title='Topics'), gridPos={ x: 0, y: 0, w: 0, h: 0 })
      .addPanel(nsqTopicsE2e, gridPos={ x: 0, y: 0, w: 24, h: 6 })
      .addPanel(nsqTopics, gridPos={ x: 0, y: 6, w: 12, h: 8 })
      .addPanel(nsqTopicsDepth, gridPos={ x: 12, y: 6, w: 12, h: 8 })
      .addPanel(grafana.row.new(title='Channels'), gridPos={ x: 0, y: 14, w: 0, h: 0 })
      .addPanel(nsqChannelE2e, gridPos={ x: 0, y: 14, w: 24, h: 6 })
      .addPanel(nsqChannelClients, gridPos={ x: 0, y: 20, w: 12, h: 8 })
      .addPanel(nsqChannelStats, gridPos={ x: 12, y: 20, w: 12, h: 8 })
      .addPanel(grafana.row.new(title='Memory'), gridPos={ x: 0, y: 28, w: 0, h: 0 })
      .addPanel(nsqHeapMemory, gridPos={ x: 0, y: 28, w: 12, h: 8 })
      .addPanel(nsqHeapObjects, gridPos={ x: 12, y: 28, w: 12, h: 8 })
      .addPanel(nsqNextGC, gridPos={ x: 0, y: 36, w: 12, h: 8 })
      .addPanel(nsqGCpause, gridPos={ x: 12, y: 36, w: 12, h: 8 }),
  },
}
