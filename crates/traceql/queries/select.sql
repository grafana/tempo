SELECT "Container" FROM spans
WHERE regexp_like("K8sClusterName", '^prod.*')
  AND "K8sNamespaceName" = 'tempo-prod'
