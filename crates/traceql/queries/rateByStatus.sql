WITH base_spans AS (
SELECT * FROM spans

)
SELECT date_bin(INTERVAL '5 minutes', to_timestamp_nanos("StartTimeUnixNano"), TIMESTAMP '1970-01-01 00:00:00') as time_bucket, "StatusCode" as status, COUNT(*) as rate FROM base_spans
GROUP BY time_bucket, status
ORDER BY time_bucket, status