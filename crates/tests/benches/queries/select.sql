-- TraceQL: {resource.k8s.cluster.name =~ "prod.*" && resource.k8s.namespace.name = "tempo-prod"} | select(resource.container)
SELECT resource."Resource"."Container"
FROM (SELECT UNNEST(t.rs) as resource FROM traces t)
WHERE resource."Resource"."K8sClusterName" LIKE 'prod%'
  AND resource."Resource"."K8sNamespaceName" = 'tempo-prod'
