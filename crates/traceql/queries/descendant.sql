SELECT * FROM spans
WHERE "ServiceName" != 'loki-querier'
  AND "ServiceName" = 'loki-gateway'
  AND "StatusCode" = 2
-- TODO: Add descendant operator (>>) logic
