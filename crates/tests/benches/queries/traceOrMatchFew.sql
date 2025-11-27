-- TraceQL: { rootServiceName = `faro-collector` && (status = error || span.http.status_code = 500)}
WITH unnest_resources AS (
  SELECT UNNEST(t.rs) as resource
  FROM traces t
  WHERE t."RootServiceName" = 'faro-collector'
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
  WHERE span."StatusCode" = 2 OR span."HttpStatusCode" = 500
)
SELECT span."SpanID"
FROM filtered_spans
