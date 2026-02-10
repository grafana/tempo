{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='imageVolumeSource', url='', help='"ImageVolumeSource represents a image volume resource."'),
  '#withPullPolicy':: d.fn(help="\"Policy for pulling OCI objects. Possible values are: Always: the kubelet always attempts to pull the reference. Container creation will fail If the pull fails. Never: the kubelet never pulls the reference and only uses a local image or artifact. Container creation will fail if the reference isn't present. IfNotPresent: the kubelet pulls if the reference isn't already present on disk. Container creation will fail if the reference isn't present and the pull fails. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.\"", args=[d.arg(name='pullPolicy', type=d.T.string)]),
  withPullPolicy(pullPolicy): { pullPolicy: pullPolicy },
  '#withReference':: d.fn(help='"Required: Image or artifact reference to be used. Behaves in the same way as pod.spec.containers[*].image. Pull secrets will be assembled in the same way as for the container image by looking up node credentials, SA image pull secrets, and pod spec image pull secrets. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets."', args=[d.arg(name='reference', type=d.T.string)]),
  withReference(reference): { reference: reference },
  '#mixin': 'ignore',
  mixin: self,
}
