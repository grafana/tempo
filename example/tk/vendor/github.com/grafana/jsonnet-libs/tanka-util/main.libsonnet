local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
{
  local this = self,

  '#':: d.pkg(
    name='tanka_util',
    url='github.com/grafana/jsonnet-libs/tanka-util/main.libsonnet',
    help=(importstr 'README.md.tmpl') % (importstr '_example.jsonnet'),
  ),

  '#k8s':: d.obj(
    |||
      `k8s` provides common utils to modify Kubernetes objects.
    |||
  ),
  k8s: (import 'k8s.libsonnet'),

  '#environment':: d.obj(
    |||
      `environment` provides a base to create an [inline Tanka
      environment](https://tanka.dev/inline-environments#inline-environments).
    |||
  ),
  environment: (import 'environment.libsonnet'),

  '#helm':: d.obj(
    |||
      `helm` allows the user to consume Helm Charts as plain Jsonnet resources.
      This implements [Helm support](https://tanka.dev/helm) for Grafana Tanka.
    |||
  ),
  helm: (import 'helm.libsonnet'),

  '#kustomize':: d.obj(
    |||
      `kustomize` allows the user to expand Kustomize manifests into plain Jsonnet resources.
      This implements [Kustomize support](https://tanka.dev/kustomize) for Grafana Tanka.
    |||
  ),
  kustomize: (import 'kustomize.libsonnet'),
}
