{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='ipAddressSpec', url='', help='"IPAddressSpec describe the attributes in an IP Address."'),
  '#parentRef':: d.obj(help='"ParentReference describes a reference to a parent object."'),
  parentRef: {
    '#withGroup':: d.fn(help='"Group is the group of the object being referenced."', args=[d.arg(name='group', type=d.T.string)]),
    withGroup(group): { parentRef+: { group: group } },
    '#withName':: d.fn(help='"Name is the name of the object being referenced."', args=[d.arg(name='name', type=d.T.string)]),
    withName(name): { parentRef+: { name: name } },
    '#withNamespace':: d.fn(help='"Namespace is the namespace of the object being referenced."', args=[d.arg(name='namespace', type=d.T.string)]),
    withNamespace(namespace): { parentRef+: { namespace: namespace } },
    '#withResource':: d.fn(help='"Resource is the resource of the object being referenced."', args=[d.arg(name='resource', type=d.T.string)]),
    withResource(resource): { parentRef+: { resource: resource } },
  },
  '#mixin': 'ignore',
  mixin: self,
}
