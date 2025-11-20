SELECT * FROM spans
WHERE ("ResourceK8sClusterName" ~ 'prod.*' AND "Name" = 'gcs.ReadRange')
