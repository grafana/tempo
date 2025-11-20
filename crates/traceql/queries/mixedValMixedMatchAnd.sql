SELECT * FROM spans
WHERE ("ResourceK8sClusterName" ~ 'prod.*' && "Name" = 'gcs.ReadRange')
