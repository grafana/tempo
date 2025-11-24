SELECT "ResourceContainer" FROM spans
WHERE ("ResourceK8sClusterName" ~ 'prod.*' AND "ResourceK8sNamespaceName" = 'tempo-prod')
