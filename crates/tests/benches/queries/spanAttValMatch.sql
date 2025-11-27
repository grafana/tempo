-- TraceQL: { span.component = `net/http` }
WITH unnest_resources AS (
  SELECT UNNEST(t.rs) as resource
  FROM traces t
),
unnest_scopespans AS (
  SELECT UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT span FROM unnest_spans
  WHERE list_contains(flatten(map_extract(attrs_to_map(span."Attrs"), 'component')), 'net/http')
)
SELECT span."SpanID"
FROM filtered_spans
