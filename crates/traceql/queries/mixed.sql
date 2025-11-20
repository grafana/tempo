SELECT * FROM spans
WHERE ((("ResourceNamespace" != '' AND "ResourceServiceName" = 'cortex-gateway') AND "DurationNano" > 50000000) AND "ResourceCluster" ~ 'prod.*')
