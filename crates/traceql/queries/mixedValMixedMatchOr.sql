SELECT * FROM spans
WHERE (list_contains(flatten(map_extract("ResourceAttrs", 'foo')), 'bar') || "Name" = 'gcs.ReadRange')
