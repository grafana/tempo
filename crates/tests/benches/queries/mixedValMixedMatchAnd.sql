-- TraceQL: { resource.k8s.cluster.name =~ `prod.*` && name = `gcs.ReadRange` }
WITH unnest_resources AS (
  SELECT UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT resource FROM unnest_resources
  WHERE resource."Resource"."K8sClusterName" LIKE 'prod%'
),
unnest_scopespans AS (
  SELECT UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT span FROM unnest_spans
  WHERE span."Name" = 'gcs.ReadRange'
)
SELECT span."SpanID"
FROM filtered_spans
