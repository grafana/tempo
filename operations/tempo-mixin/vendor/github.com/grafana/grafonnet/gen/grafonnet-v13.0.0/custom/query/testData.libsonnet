local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#withDatasource':: d.func.new(
    'Set the datasource for this query.',
    args=[
      d.arg('value', d.T.string),
    ]
  ),
  withDatasource(): {
    datasource+: {
      type: 'datasource',
      uid: 'grafana',
    },
  },
  '#withDatasourceMixin':: { ignore: true },
}
