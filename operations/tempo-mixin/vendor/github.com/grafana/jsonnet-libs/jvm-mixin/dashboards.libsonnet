local dashboard = import 'jvm_rev1.libsonnet';

{
  grafanaDashboards+:: {
    'jvm-dashboard.json': dashboard,
  },
}
