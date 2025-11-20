SELECT * FROM spans
WHERE ("rootServiceName" = 'doesntexist' && ("StatusCode" = 2 || "HttpStatusCode" = 500))
