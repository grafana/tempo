-- Flattened Spans View
-- This view unnests the nested trace structure and presents one row per span
-- Each row contains all span fields with Attrs converted to a Map for convenient access
-- Usage: SELECT * FROM spans WHERE Name = 'query-product'
-- Access attributes: SELECT TraceID, Name, Attrs['http.method'] FROM spans

CREATE OR REPLACE VIEW spans AS
WITH unnest_resources AS (
    SELECT
        t."TraceID",
        UNNEST(t.rs) as resource
    FROM traces t
),
unnest_spansets AS (
    SELECT
        "TraceID",
        UNNEST(resource.ss) as spanset
    FROM unnest_resources
),
unnest_spans AS (
    SELECT
        "TraceID",
        UNNEST(spanset."Spans") as span
    FROM unnest_spansets
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
    span."DedicatedAttributes" AS "DedicatedAttributes"
FROM unnest_spans
