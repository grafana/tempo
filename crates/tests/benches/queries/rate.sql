-- TraceQL: { } | rate()
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
unnest_scopespans AS (
  SELECT "TraceID", UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
)
SELECT date_bin(INTERVAL '1 minute', to_timestamp_nanos(CAST(span."StartTimeUnixNano" AS BIGINT)), TIMESTAMP '1970-01-01 00:00:00') as time_bucket, COUNT(*) as rate FROM unnest_spans
GROUP BY time_bucket
ORDER BY time_bucket
