-- TraceQL: { name = `distributor.ConsumeTraces` }
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
  WHERE span."Name" = 'distributor.ConsumeTraces'
)
SELECT span."SpanID"
FROM filtered_spans
