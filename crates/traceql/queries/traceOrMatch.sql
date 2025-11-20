SELECT * FROM spans
WHERE ("rootServiceName" = 'tempo-distributor' && ("StatusCode" = 2 || "HttpStatusCode" = 500))
