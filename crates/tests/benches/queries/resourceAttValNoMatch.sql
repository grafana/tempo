-- TraceQL: { resource.module.path = `does-not-exit-6c2408325a45` }
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT * FROM unnest_resources
  WHERE list_contains(flatten(map_extract(attrs_to_map(resource."Resource"."Attrs"), 'module.path')), 'does-not-exit-6c2408325a45')
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT "TraceID", UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
)
SELECT "TraceID", span."SpanID"
FROM unnest_spans
