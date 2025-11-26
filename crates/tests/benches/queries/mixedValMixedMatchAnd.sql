-- TraceQL: { resource.k8s.cluster.name =~ `prod.*` && name = `gcs.ReadRange` }
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT * FROM unnest_resources
  WHERE resource."Resource"."K8sClusterName" LIKE 'prod%'
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT "TraceID", UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT * FROM unnest_spans
  WHERE span."Name" = 'gcs.ReadRange'
)
SELECT "TraceID", span."SpanID"
FROM filtered_spans
