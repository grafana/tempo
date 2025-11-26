WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT * FROM unnest_resources
  WHERE resource."Resource"."K8sClusterName" ~ 'prod.*' AND resource."Resource"."K8sNamespaceName" = 'tempo-prod'
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT "TraceID", resource, UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
)
SELECT resource."Resource"."Container"
FROM unnest_spans
