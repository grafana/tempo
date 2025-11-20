SELECT * FROM spans
WHERE list_contains(flatten(map_extract("Attrs", 'bloom')), 'does-not-exit-6c2408325a45')
