SELECT * FROM spans
WHERE list_contains(flatten(map_extract("Attrs", 'component')), 'net/http')
