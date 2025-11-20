SELECT * FROM spans
WHERE list_contains(flatten(map_extract("ResourceAttrs", 'module.path')), 'does-not-exit-6c2408325a45')
