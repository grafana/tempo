WITH base_spans AS (
SELECT * FROM spans

)
SELECT COUNT(*) as count FROM base_spans
 WHERE count > 1
