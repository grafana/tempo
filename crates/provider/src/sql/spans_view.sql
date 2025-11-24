CREATE OR REPLACE VIEW spans AS
WITH unnest_resources AS (
    SELECT
        t."TraceID",
        UNNEST(t.rs) as resource
    FROM traces t
),
unnest_scopespans AS (
    SELECT
        "TraceID",
        resource,
        UNNEST(resource.ss) as scopespans
    FROM unnest_resources
),
unnest_spans AS (
    SELECT
        "TraceID",
        resource,
        UNNEST(scopespans."Spans") as span
    FROM unnest_scopespans
)
SELECT
    "TraceID",
    span."SpanID" AS "SpanID",
    span."ParentSpanID" AS "ParentSpanID",
    span."ParentID" AS "ParentID",
    span."NestedSetLeft" AS "NestedSetLeft",
    span."NestedSetRight" AS "NestedSetRight",
    span."Name" AS "Name",
    span."Kind" AS "Kind",
    span."TraceState" AS "TraceState",
    span."StartTimeUnixNano" AS "StartTimeUnixNano",
    span."DurationNano" AS "DurationNano",
    span."StatusCode" AS "StatusCode",
    span."StatusMessage" AS "StatusMessage",
    attrs_to_map(span."Attrs") AS "Attrs",
    span."DroppedAttributesCount" AS "DroppedAttributesCount",
    span."Events" AS "Events",
    span."DroppedEventsCount" AS "DroppedEventsCount",
    span."Links" AS "Links",
    span."DroppedLinksCount" AS "DroppedLinksCount",
    span."HttpMethod" AS "HttpMethod",
    span."HttpUrl" AS "HttpUrl",
    span."HttpStatusCode" AS "HttpStatusCode",
    span."DedicatedAttributes" AS "DedicatedAttributes",
    -- Resource attributes
    --attrs_to_map(resource."Resource"."Attrs") AS "ResourceAttrs",
    --resource."Resource"."ServiceName" AS "ResourceServiceName",
    --resource."Resource"."Cluster" AS "ResourceCluster",
    --resource."Resource"."Namespace" AS "ResourceNamespace",
    --resource."Resource"."Pod" AS "ResourcePod",
    --resource."Resource"."Container" AS "ResourceContainer",
    --resource."Resource"."K8sClusterName" AS "ResourceK8sClusterName",
    --resource."Resource"."K8sNamespaceName" AS "ResourceK8sNamespaceName",
    --resource."Resource"."K8sPodName" AS "ResourceK8sPodName",
    --resource."Resource"."K8sContainerName" AS "ResourceK8sContainerName",
    --resource."Resource"."DedicatedAttributes" AS "ResourceDedicatedAttributes"
FROM unnest_spans
