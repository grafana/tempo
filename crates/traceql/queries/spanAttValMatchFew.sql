SELECT * FROM spans
WHERE array_to_string(flatten(map_extract("Attrs", 'component')), ',') ~ 'database/sql'
