local grafana = import 'grafana/grafana.libsonnet';

{
  tempo(url='http://query-frontend'):
    grafana.datasource.new(
      'Tempo',
      url,
      'tempo',
      default=true,
    ),
  prometheus:
    grafana.datasource.new(
      'Prometheus',
      'http://prometheus/prometheus',
      'prometheus',
    ),
}
