SELECT * FROM spans
WHERE "ResourceServiceName" = 'loki-querier'
UNION
SELECT * FROM spans
WHERE "ResourceServiceName" = 'loki-gateway'
