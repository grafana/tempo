-- TraceQL: {resource.k8s.cluster.name =~ "prod.*" && resource.k8s.namespace.name = "hosted-grafana" && resource.k8s.container.name="hosted-grafana-gateway" && name = "httpclient/grafana" && span.http.status_code = 200 && duration > 20ms}
WITH unnest_resources AS (
  SELECT UNNEST(t.rs) as resource
  FROM traces t
),
filtered_resources AS (
  SELECT resource FROM unnest_resources
  WHERE resource."Resource"."K8sClusterName" LIKE 'prod%'
    AND resource."Resource"."K8sNamespaceName" = 'hosted-grafana'
    AND resource."Resource"."K8sContainerName" = 'hosted-grafana-gateway'
),
unnest_scopespans AS (
  SELECT UNNEST(resource.ss) as scopespans
  FROM filtered_resources
),
unnest_spans AS (
  SELECT UNNEST(scopespans."Spans") as span
  FROM unnest_scopespans
),
filtered_spans AS (
  SELECT span FROM unnest_spans
  WHERE span."Name" = 'httpclient/grafana'
    AND span."HttpStatusCode" = 200
    AND span."DurationNano" > 20000000
)
SELECT span."SpanID"
FROM filtered_spans
