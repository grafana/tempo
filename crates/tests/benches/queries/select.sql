-- TraceQL: {resource.k8s.cluster.name =~ "prod.*" && resource.k8s.namespace.name = "tempo-prod"} | select(resource.container)
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT * FROM unnest_resources
  WHERE resource."Resource"."K8sClusterName" LIKE 'prod%'
    AND resource."Resource"."K8sNamespaceName" = 'tempo-prod'
),
unnest_scopespans AS (
  SELECT "TraceID", resource."Resource"."Container" as container, UNNEST(resource.ss) as scopespans
  FROM filtered_resources
)
SELECT DISTINCT "TraceID", container
FROM unnest_scopespans
