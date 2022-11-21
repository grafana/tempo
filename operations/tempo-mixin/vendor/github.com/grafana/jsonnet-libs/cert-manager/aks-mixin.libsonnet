{
  // This is a mixin which adds a namespaceSelector to exclude control-plane namespaces.
  // Ref: https://docs.microsoft.com/en-us/azure/aks/faq#can-i-use-admission-controller-webhooks-on-aks
  // This selector gets added automatically by AKS after applying, by using this mixin, we can prevent a diff.
  local webhooks = super.labeled.validating_webhook_configuration_cert_manager_webhook.webhooks,
  labeled+: {
    validating_webhook_configuration_cert_manager_webhook+: {
      webhooks: [
        webhook {
          namespaceSelector+: {
            matchExpressions+: [
              {
                key: 'control-plane',
                operator: 'DoesNotExist',
              },
            ],
          },
        }
        for webhook in webhooks
      ],
    },
  },
}
