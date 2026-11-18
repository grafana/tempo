{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='nfsVolumeSource', url='', help='"Represents an NFS mount that lasts the lifetime of a pod. NFS volumes do not support ownership management or SELinux relabeling."'),
  '#withPath':: d.fn(help='"path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs"', args=[d.arg(name='path', type=d.T.string)]),
  withPath(path): { path: path },
  '#withReadOnly':: d.fn(help='"readOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs"', args=[d.arg(name='readOnly', type=d.T.boolean)]),
  withReadOnly(readOnly): { readOnly: readOnly },
  '#withServer':: d.fn(help='"server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs"', args=[d.arg(name='server', type=d.T.string)]),
  withServer(server): { server: server },
  '#mixin': 'ignore',
  mixin: self,
}
