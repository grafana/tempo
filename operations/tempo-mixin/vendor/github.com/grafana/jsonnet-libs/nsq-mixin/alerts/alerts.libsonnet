{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'nsq',
        rules: [
          {
            alert: 'NsqTopicDepthIncreasing',
            expr: |||
              sum by (topic) (nsq_topic_depth) > %(alertsCriticalTopicDepth)s
            ||| % $._config,
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Topic depth is increasing.',
              description: |||
                Topic {{ $labels.topic }} depth is higher than %(alertsCriticalTopicDepth)s. The currect queue is {{ $value }}.
              ||| % $._config,
            },
          },
          {
            alert: 'NsqChannelDepthIncreasing',
            expr: |||
              sum by (topic) (nsq_topic_channel_backend_depth) > %(alertsCriticalChannelDepth)s
            ||| % $._config,
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Topic channel depth is increasing.',
              description: |||
                Channel {{ $labels.channel }} depth in topic {{ $labels.topic }} is higher than %(alertsCriticalChannelDepth)s. The currect queue is {{ $value }}.
              ||| % $._config,
            },
          },
        ],
      },
    ],
  },
}
