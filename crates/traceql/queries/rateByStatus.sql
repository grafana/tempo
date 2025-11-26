WITH base_spans AS (
WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT "TraceID", resource, UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
)
SELECT "TraceID" AS "TraceID", span."SpanID" AS "SpanID", span."Name" AS "Name", span."Kind" AS "Kind", span."ParentSpanID" AS "ParentSpanID", span."StartTimeUnixNano" AS "StartTimeUnixNano", span."DurationNano" AS "DurationNano", span."StatusCode" AS "StatusCode", span."HttpMethod" AS "HttpMethod", span."HttpUrl" AS "HttpUrl", span."HttpStatusCode" AS "HttpStatusCode", attrs_to_map(span."Attrs") AS "Attrs", resource."Resource"."ServiceName" AS "ServiceName", resource."Resource"."Cluster" AS "Cluster", resource."Resource"."Namespace" AS "Namespace", resource."Resource"."Pod" AS "Pod", resource."Resource"."Container" AS "Container", resource."Resource"."K8sClusterName" AS "K8sClusterName", resource."Resource"."K8sNamespaceName" AS "K8sNamespaceName", resource."Resource"."K8sPodName" AS "K8sPodName", resource."Resource"."K8sContainerName" AS "K8sContainerName", attrs_to_map(resource."Resource"."Attrs") AS "ResourceAttrs", span."NestedSetLeft" AS "NestedSetLeft", span."NestedSetRight" AS "NestedSetRight"
FROM unnest_spans
)
SELECT date_bin(INTERVAL '5 minutes', to_timestamp_nanos(CAST("StartTimeUnixNano" AS BIGINT)), TIMESTAMP '1970-01-01 00:00:00') as time_bucket, "StatusCode", COUNT(*) as rate FROM base_spans
GROUP BY time_bucket, "StatusCode"
ORDER BY time_bucket, "StatusCode"
