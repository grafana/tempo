SELECT spans.* FROM spans
INNER JOIN spans root ON root."TraceID" = spans."TraceID"
  AND (root."ParentSpanID" = '' OR root."ParentSpanID" IS NULL)
WHERE (root."ResourceServiceName" = 'doesntexist' AND (spans."StatusCode" = 2 OR spans."HttpStatusCode" = 500))
