local grafana = import 'grafana/grafana.libsonnet';

{
  tempo(url='http://query-frontend'):
    grafana.datasource.new(
      'Tempo',
      url,
      'tempo',
    ),
  prometheus:
    grafana.datasource.new(
      'Prometheus',
      'http://prometheus/prometheus',
      'prometheus',
      true,
    ) +
    grafana.datasource.withHttpMethod('POST'),
}
