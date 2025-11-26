-- TraceQL: { } | count() > 1
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
unnest_scopespans AS (
  SELECT "TraceID", UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT "TraceID", UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
)
SELECT "TraceID", COUNT(*) as count
FROM unnest_spans
GROUP BY "TraceID"
HAVING count > 1
