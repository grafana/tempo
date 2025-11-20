SELECT * FROM spans
WHERE ("ResourceAttrs"['foo'] = 'bar' || "Name" = 'gcs.ReadRange')
