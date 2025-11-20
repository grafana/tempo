SELECT * FROM spans
WHERE ((((("ResourceK8sClusterName" ~ 'prod.*' && "ResourceK8sNamespaceName" = 'hosted-grafana') && "ResourceK8sContainerName" = 'hosted-grafana-gateway') && "Name" = 'httpclient/grafana') && "HttpStatusCode" = 200) && "DurationNano" > 20000000)
