SELECT * FROM spans
WHERE list_contains(flatten(map_extract("ResourceAttrs", 'opencensus.exporterversion')), 'Jaeger-Go-2.30.0')
