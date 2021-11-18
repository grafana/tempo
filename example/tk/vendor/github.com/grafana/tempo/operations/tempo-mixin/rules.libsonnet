local utils = import 'mixin-utils/utils.libsonnet';

{
  prometheusRules+:: {
    groups+: [{
      name: 'tempo_rules',
      rules:
        utils.histogramRules('tempo_request_duration_seconds', $._config.job_selectors + ['route']),
    }],
  },
}
