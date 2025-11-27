-- TraceQL: { resource.service.name = `loki-querier` } || { resource.service.name = `loki-gateway` }
WITH unnest_resources AS (
  SELECT UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT resource FROM unnest_resources
  WHERE resource."Resource"."ServiceName" IN ('loki-querier', 'loki-gateway')
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
