{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='clusterTrustBundleSpec', url='', help='"ClusterTrustBundleSpec contains the signer and trust anchors."'),
  '#withSignerName':: d.fn(help="\"signerName indicates the associated signer, if any.\\n\\nIn order to create or update a ClusterTrustBundle that sets signerName, you must have the following cluster-scoped permission: group=certificates.k8s.io resource=signers resourceName=\u003cthe signer name\u003e verb=attest.\\n\\nIf signerName is not empty, then the ClusterTrustBundle object must be named with the signer name as a prefix (translating slashes to colons). For example, for the signer name `example.com/foo`, valid ClusterTrustBundle object names include `example.com:foo:abc` and `example.com:foo:v1`.\\n\\nIf signerName is empty, then the ClusterTrustBundle object's name must not have such a prefix.\\n\\nList/watch requests for ClusterTrustBundles can filter on this field using a `spec.signerName=NAME` field selector.\"", args=[d.arg(name='signerName', type=d.T.string)]),
  withSignerName(signerName): { signerName: signerName },
  '#withTrustBundle':: d.fn(help='"trustBundle contains the individual X.509 trust anchors for this bundle, as PEM bundle of PEM-wrapped, DER-formatted X.509 certificates.\\n\\nThe data must consist only of PEM certificate blocks that parse as valid X.509 certificates.  Each certificate must include a basic constraints extension with the CA bit set.  The API server will reject objects that contain duplicate certificates, or that use PEM block headers.\\n\\nUsers of ClusterTrustBundles, including Kubelet, are free to reorder and deduplicate certificate blocks in this file according to their own logic, as well as to drop PEM block headers and inter-block data."', args=[d.arg(name='trustBundle', type=d.T.string)]),
  withTrustBundle(trustBundle): { trustBundle: trustBundle },
  '#mixin': 'ignore',
  mixin: self,
}
