SELECT * FROM spans
WHERE ((list_contains(flatten(map_extract(\"Attrs\", 'foo')), 'a') AND list_contains(flatten(map_extract(\"Attrs\", 'bar')), 'a')) OR (list_contains(flatten(map_extract(\"Attrs\", 'foo')), 'b') AND NOT list_contains(flatten(map_extract(\"Attrs\", 'bar')), 'a')))
