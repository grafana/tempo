WITH base_spans AS (
SELECT * FROM spans

)
SELECT status, COUNT(*) as count FROM base_spans
GROUP BY status
 WHERE count > 1
