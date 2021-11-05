local get = import '../main.libsonnet';

get {
  _config+:: {
    namespace: 'enterprise-traces-test0',
    bucket: 'enterprise-traces-test',
    backend: 's3',
  },
}
