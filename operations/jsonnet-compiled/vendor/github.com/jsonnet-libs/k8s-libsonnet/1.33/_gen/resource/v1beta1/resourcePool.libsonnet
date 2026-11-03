{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourcePool', url='', help='"ResourcePool describes the pool that ResourceSlices belong to."'),
  '#withGeneration':: d.fn(help='"Generation tracks the change in a pool over time. Whenever a driver changes something about one or more of the resources in a pool, it must change the generation in all ResourceSlices which are part of that pool. Consumers of ResourceSlices should only consider resources from the pool with the highest generation number. The generation may be reset by drivers, which should be fine for consumers, assuming that all ResourceSlices in a pool are updated to match or deleted.\\n\\nCombined with ResourceSliceCount, this mechanism enables consumers to detect pools which are comprised of multiple ResourceSlices and are in an incomplete state."', args=[d.arg(name='generation', type=d.T.integer)]),
  withGeneration(generation): { generation: generation },
  '#withName':: d.fn(help='"Name is used to identify the pool. For node-local devices, this is often the node name, but this is not required.\\n\\nIt must not be longer than 253 characters and must consist of one or more DNS sub-domains separated by slashes. This field is immutable."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withResourceSliceCount':: d.fn(help='"ResourceSliceCount is the total number of ResourceSlices in the pool at this generation number. Must be greater than zero.\\n\\nConsumers can use this to check whether they have seen all ResourceSlices belonging to the same pool."', args=[d.arg(name='resourceSliceCount', type=d.T.integer)]),
  withResourceSliceCount(resourceSliceCount): { resourceSliceCount: resourceSliceCount },
  '#mixin': 'ignore',
  mixin: self,
}
