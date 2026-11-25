local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  local withDatasourceStub = {
    '#withDatasource':: d.func.new(
      'Set the datasource for this query.',
      args=[
        d.arg('value', d.T.string),
      ]
    ),
    withDatasource(value): {
      datasource+: {
        type: 'cloudwatch',
        uid: value,
      },
    },
    '#withDatasourceMixin':: { ignore: true },
  },

  CloudWatchAnnotationQuery+: withDatasourceStub,
  CloudWatchLogsQuery+: withDatasourceStub,
  CloudWatchMetricsQuery+: withDatasourceStub,
}
