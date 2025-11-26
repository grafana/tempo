WITH unnest_resources AS (
  SELECT t."TraceID", UNNEST(t.rs) as resource
  FROM traces t
  WHERE t."RootServiceName" = 'tempo-distributor'
),
unnest_scopespans AS (
  SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
  FROM unnest_resources
),
unnest_spans AS (
  SELECT "TraceID", resource, UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT * FROM unnest_spans
  WHERE span."StatusCode" = 2 AND span."HttpStatusCode" = 500 AND (span."StatusCode" = 2 OR span."HttpStatusCode" = 500)
)
SELECT "TraceID" AS "TraceID", span."SpanID" AS "SpanID", span."Name" AS "Name", span."Kind" AS "Kind", span."ParentSpanID" AS "ParentSpanID", span."StartTimeUnixNano" AS "StartTimeUnixNano", span."DurationNano" AS "DurationNano", span."StatusCode" AS "StatusCode", span."HttpMethod" AS "HttpMethod", span."HttpUrl" AS "HttpUrl", span."HttpStatusCode" AS "HttpStatusCode", attrs_to_map(span."Attrs") AS "Attrs", resource."Resource"."ServiceName" AS "ServiceName", resource."Resource"."Cluster" AS "Cluster", resource."Resource"."Namespace" AS "Namespace", resource."Resource"."Pod" AS "Pod", resource."Resource"."Container" AS "Container", resource."Resource"."K8sClusterName" AS "K8sClusterName", resource."Resource"."K8sNamespaceName" AS "K8sNamespaceName", resource."Resource"."K8sPodName" AS "K8sPodName", resource."Resource"."K8sContainerName" AS "K8sContainerName", attrs_to_map(resource."Resource"."Attrs") AS "ResourceAttrs", span."NestedSetLeft" AS "NestedSetLeft", span."NestedSetRight" AS "NestedSetRight"
FROM filtered_spans
