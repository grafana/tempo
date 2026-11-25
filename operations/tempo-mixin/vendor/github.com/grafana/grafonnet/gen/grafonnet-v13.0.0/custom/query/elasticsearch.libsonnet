local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  bucketAggs+: { '#': { help: '', name: 'bucketAggs' } },
  metrics+: { '#': { help: '', name: 'metrics' } },

  '#withDatasource':: d.func.new(
    'Set the datasource for this query.',
    args=[
      d.arg('value', d.T.string),
    ]
  ),
  withDatasource(value): {
    datasource+: {
      type: 'elasticsearch',
      uid: value,
    },
  },
  '#withDatasourceMixin':: { ignore: true },
}
