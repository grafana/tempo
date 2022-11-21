# cert-manager

This jsonnet lib renders the cert-manager Helm chart with a few Grafana specific overrides.

It depends on the helmraiser functionality available in tanka>=0.12.0-alpha1.

## AKS

`aks-mixin.libsonnet` is a mixin which adds a namespaceSelector to exclude control-plane namespaces. This selector gets
added automatically by AKS after applying, by using this mixin, we can prevent a diff.

References:

* https://github.com/Azure/AKS/issues/1771
* https://docs.microsoft.com/en-us/azure/aks/faq#can-i-use-admission-controller-webhooks-on-aks
