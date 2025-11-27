-- TraceQL: { resource.service.name != `loki-querier` } >> { resource.service.name = `loki-gateway` && status = error }
WITH unnest_resources_parent AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources_parent AS (
  SELECT "TraceID", resource FROM unnest_resources_parent
  WHERE resource."Resource"."ServiceName" != 'loki-querier'
),
unnest_scopespans_parent AS (
  SELECT "TraceID", UNNEST(resource.ss) as scopespans
  FROM filtered_resources_parent
),
parent_spans AS (
  SELECT "TraceID", UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans_parent
),
unnest_resources_child AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources_child AS (
  SELECT "TraceID", resource FROM unnest_resources_child
  WHERE resource."Resource"."ServiceName" = 'loki-gateway'
),
unnest_scopespans_child AS (
  SELECT "TraceID", UNNEST(resource.ss) as scopespans
  FROM filtered_resources_child
),
child_spans AS (
  SELECT "TraceID", UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans_child
),
filtered_child_spans AS (
  SELECT "TraceID", span FROM child_spans
  WHERE span."StatusCode" = 2
)
SELECT DISTINCT p.span."SpanID"
FROM parent_spans p
INNER JOIN filtered_child_spans c
  ON p."TraceID" = c."TraceID"
  AND c.span."NestedSetLeft" > p.span."NestedSetLeft"
  AND c.span."NestedSetRight" < p.span."NestedSetRight"
