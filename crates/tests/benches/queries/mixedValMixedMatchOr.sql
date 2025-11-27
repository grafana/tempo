-- TraceQL: { resource.foo = `bar` || name = `gcs.ReadRange` }
WITH unnest_resources AS (
  SELECT UNNEST(t.rs) as resource
  FROM traces t
),
unnest_scopespans AS (
  SELECT resource, UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT resource, UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT span FROM unnest_spans
  WHERE list_contains(flatten(map_extract(attrs_to_map(resource."Resource"."Attrs"), 'foo')), 'bar')
     OR span."Name" = 'gcs.ReadRange'
)
SELECT span."SpanID"
FROM filtered_spans
