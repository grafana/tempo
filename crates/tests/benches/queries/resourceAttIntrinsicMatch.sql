-- TraceQL: { resource.service.name = `tempo-gateway` }
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT * FROM unnest_resources
  WHERE resource."Resource"."ServiceName" = 'tempo-gateway'
),
unnest_scopespans AS (
  SELECT "TraceID", UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT "TraceID", UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
)
SELECT "TraceID", span."SpanID"
FROM unnest_spans
