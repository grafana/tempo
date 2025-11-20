SELECT * FROM spans
WHERE ((((("ResourceK8sClusterName" ~ 'prod.*' AND "ResourceK8sNamespaceName" = 'hosted-grafana') AND "ResourceK8sContainerName" = 'hosted-grafana-gateway') AND "Name" = 'httpclient/grafana') AND "HttpStatusCode" = 200) AND "DurationNano" > 20000000)
