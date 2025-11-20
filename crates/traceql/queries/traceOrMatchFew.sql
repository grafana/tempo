SELECT * FROM spans
WHERE ("rootServiceName" = 'faro-collector' && ("StatusCode" = 2 || "HttpStatusCode" = 500))
