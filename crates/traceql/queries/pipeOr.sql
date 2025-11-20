SELECT * FROM spans
WHERE "ServiceName" = 'loki-querier'
UNION
SELECT * FROM spans
WHERE "ServiceName" = 'loki-gateway'
