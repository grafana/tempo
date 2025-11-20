SELECT * FROM spans
WHERE ((("ResourceNamespace" != '' && "ResourceServiceName" = 'cortex-gateway') && "DurationNano" > 50000000) && "ResourceCluster" ~ 'prod.*')
