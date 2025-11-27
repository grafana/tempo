-- TraceQL: { resource.service.name = `does-not-exit-6c2408325a45` }
WITH unnest_resources AS (
  SELECT UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT resource FROM unnest_resources
  WHERE resource."Resource"."ServiceName" = 'does-not-exit-6c2408325a45'
),
unnest_scopespans AS (
  SELECT UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
)
SELECT span."SpanID"
FROM unnest_spans
