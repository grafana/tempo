-- TraceQL: {resource.namespace!="" && resource.service.name="cortex-gateway" && duration>50ms && resource.cluster=~"prod.*"}
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT * FROM unnest_resources
  WHERE resource."Resource"."Namespace" IS NOT NULL
    AND resource."Resource"."Namespace" != ''
    AND resource."Resource"."ServiceName" = 'cortex-gateway'
    AND resource."Resource"."Cluster" LIKE 'prod%'
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
  WHERE span."DurationNano" > 50000000
)
SELECT "TraceID", span."SpanID"
FROM filtered_spans
