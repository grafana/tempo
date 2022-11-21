local dashboard = import 'minio_v1.libsonnet';
{
  grafanaDashboards+: {
    'minio-dashboardv1.json': dashboard,
  },
}
