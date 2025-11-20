SELECT * FROM spans
WHERE (list_contains(flatten(map_extract("ResourceAttrs", 'foo')), 'bar') OR "Name" = 'gcs.ReadRange')
