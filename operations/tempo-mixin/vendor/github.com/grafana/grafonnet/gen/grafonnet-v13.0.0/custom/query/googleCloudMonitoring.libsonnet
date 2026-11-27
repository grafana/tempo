local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#withDatasource':: d.func.new(
    'Set the datasource for this query.',
    args=[
      d.arg('value', d.T.string),
    ]
  ),
  withDatasource(value): {
    datasource+: {
      type: 'cloud-monitoring',
      uid: value,
    },
  },
  '#withDatasourceMixin':: { ignore: true },
}
