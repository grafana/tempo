-- TraceQL: { resource.foo = `bar` || name = `gcs.ReadRange` }
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT "TraceID", resource, UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT * FROM unnest_spans
  WHERE list_contains(flatten(map_extract(attrs_to_map(resource."Resource"."Attrs"), 'foo')), 'bar')
     OR span."Name" = 'gcs.ReadRange'
)
SELECT "TraceID", span."SpanID"
FROM filtered_spans
