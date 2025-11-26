-- TraceQL: { rootServiceName = `faro-collector` && (status = error || span.http.status_code = 500)}
WITH filtered_traces AS (
  SELECT * FROM traces t
  WHERE t."RootServiceName" = 'faro-collector'
),
unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM filtered_traces t
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT "TraceID", UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT * FROM unnest_spans
  WHERE span."StatusCode" = 2 OR span."HttpStatusCode" = 500
)
SELECT "TraceID", span."SpanID"
FROM filtered_spans
