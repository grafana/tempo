WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT * FROM unnest_resources
  WHERE resource."Resource"."ServiceName" != 'loki-querier'
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT "TraceID", resource, UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
), parent_spans AS (
  SELECT * FROM unnest_spans
)
, child_unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
child_filtered_resources AS (
  SELECT * FROM child_unnest_resources
  WHERE resource."Resource"."ServiceName" = 'loki-gateway'
),
child_unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM child_filtered_resources
),
child_unnest_spans AS (
  SELECT "TraceID", resource, UNNEST(scopespans."Spans") as span
  FROM child_unnest_scopespans
),
child_filtered_spans AS (
  SELECT * FROM child_unnest_spans
  WHERE span."StatusCode" = 2
),
child_spans AS (
  SELECT "TraceID", span."SpanID", span."Name", span."NestedSetLeft", span."NestedSetRight" FROM child_filtered_spans
)
SELECT child_spans.* FROM parent_spans
INNER JOIN child_spans
  ON child_spans."TraceID" = parent_spans."TraceID"
  AND child_spans."NestedSetLeft" > parent_spans."NestedSetLeft"
  AND child_spans."NestedSetRight" < parent_spans."NestedSetRight"

