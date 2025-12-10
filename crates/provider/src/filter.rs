use chrono::DateTime;
use datafusion::logical_expr::{Expr, Operator};
use datafusion::scalar::ScalarValue;
use storage::DiscoveredBlock;
use tracing::{debug, warn};
use vparquet4::SpanFilter;

/// The column name for the start time in nanoseconds
pub const START_TIME_COLUMN: &str = "StartTimeUnixNano";

/// Represents a time range extracted from filter expressions
#[derive(Debug, Clone, PartialEq)]
pub struct TimeRange {
    /// Minimum time in nanoseconds (inclusive), None means no lower bound
    pub min_nanos: Option<i64>,
    /// Maximum time in nanoseconds (inclusive), None means no upper bound
    pub max_nanos: Option<i64>,
}

impl TimeRange {
    /// Create a new unbounded time range
    pub fn unbounded() -> Self {
        Self {
            min_nanos: None,
            max_nanos: None,
        }
    }

    /// Check if this time range is unbounded (no constraints)
    pub fn is_unbounded(&self) -> bool {
        self.min_nanos.is_none() && self.max_nanos.is_none()
    }

    /// Check if this time range overlaps with a block's time range
    /// Block times are in RFC3339 format (e.g., "2024-01-01T00:00:00Z")
    pub fn overlaps_with_block(&self, block: &DiscoveredBlock) -> bool {
        // Parse block start and end times
        let block_start_nanos = match DateTime::parse_from_rfc3339(&block.start_time) {
            Ok(dt) => dt.timestamp_nanos_opt().unwrap_or(i64::MIN),
            Err(e) => {
                warn!(
                    "Failed to parse block start_time '{}': {}, including block",
                    block.start_time, e
                );
                return true; // Include blocks with unparseable times to be safe
            }
        };

        let block_end_nanos = match DateTime::parse_from_rfc3339(&block.end_time) {
            Ok(dt) => dt.timestamp_nanos_opt().unwrap_or(i64::MAX),
            Err(e) => {
                warn!(
                    "Failed to parse block end_time '{}': {}, including block",
                    block.end_time, e
                );
                return true; // Include blocks with unparseable times to be safe
            }
        };

        // Check if the time ranges overlap
        // Query range: [min_nanos, max_nanos]
        // Block range: [block_start_nanos, block_end_nanos]
        // They overlap if: query_min <= block_end AND query_max >= block_start

        let query_min = self.min_nanos.unwrap_or(i64::MIN);
        let query_max = self.max_nanos.unwrap_or(i64::MAX);

        query_min <= block_end_nanos && query_max >= block_start_nanos
    }

    /// Merge this time range with another, taking the intersection
    pub fn intersect(&mut self, other: &TimeRange) {
        // Take the maximum of the minimums
        self.min_nanos = match (self.min_nanos, other.min_nanos) {
            (Some(a), Some(b)) => Some(a.max(b)),
            (Some(a), None) => Some(a),
            (None, Some(b)) => Some(b),
            (None, None) => None,
        };

        // Take the minimum of the maximums
        self.max_nanos = match (self.max_nanos, other.max_nanos) {
            (Some(a), Some(b)) => Some(a.min(b)),
            (Some(a), None) => Some(a),
            (None, Some(b)) => Some(b),
            (None, None) => None,
        };
    }
}

/// Extract time ranges from filter expressions
/// This function recursively walks the expression tree looking for filters on StartTimeUnixNano
pub fn extract_time_range_from_expr(expr: &Expr) -> TimeRange {
    debug!("Analyzing filter expression: {:?}", expr);

    match expr {
        // Handle binary expressions like: StartTimeUnixNano > 1000
        Expr::BinaryExpr(binary_expr) => {
            let left = &binary_expr.left;
            let right = &binary_expr.right;
            let op = &binary_expr.op;

            // Check if this is a filter on StartTimeUnixNano (case-insensitive)
            if let Expr::Column(col) = left.as_ref() {
                if col.name.eq_ignore_ascii_case(START_TIME_COLUMN) {
                    if let Expr::Literal(ScalarValue::Int64(Some(value)), _) = right.as_ref() {
                        let range = match op {
                            Operator::Gt => TimeRange {
                                min_nanos: Some(value + 1),
                                max_nanos: None,
                            },
                            Operator::GtEq => TimeRange {
                                min_nanos: Some(*value),
                                max_nanos: None,
                            },
                            Operator::Lt => TimeRange {
                                min_nanos: None,
                                max_nanos: Some(value - 1),
                            },
                            Operator::LtEq => TimeRange {
                                min_nanos: None,
                                max_nanos: Some(*value),
                            },
                            Operator::Eq => TimeRange {
                                min_nanos: Some(*value),
                                max_nanos: Some(*value),
                            },
                            _ => TimeRange::unbounded(),
                        };
                        debug!(
                            "Extracted time range from {} {} {}: {:?}",
                            col.name, op, value, range
                        );
                        return range;
                    }
                }
            }
            // Also check reversed order: 1000 < StartTimeUnixNano (case-insensitive)
            else if let Expr::Column(col) = right.as_ref() {
                if col.name.eq_ignore_ascii_case(START_TIME_COLUMN) {
                    if let Expr::Literal(ScalarValue::Int64(Some(value)), _) = left.as_ref() {
                        let range = match op {
                            Operator::Lt => TimeRange {
                                min_nanos: Some(value + 1),
                                max_nanos: None,
                            },
                            Operator::LtEq => TimeRange {
                                min_nanos: Some(*value),
                                max_nanos: None,
                            },
                            Operator::Gt => TimeRange {
                                min_nanos: None,
                                max_nanos: Some(value - 1),
                            },
                            Operator::GtEq => TimeRange {
                                min_nanos: None,
                                max_nanos: Some(*value),
                            },
                            Operator::Eq => TimeRange {
                                min_nanos: Some(*value),
                                max_nanos: Some(*value),
                            },
                            _ => TimeRange::unbounded(),
                        };
                        debug!(
                            "Extracted time range from {} {} {}: {:?}",
                            value, op, col.name, range
                        );
                        return range;
                    }
                }
            }

            // Handle AND: intersect the time ranges
            if matches!(op, Operator::And) {
                let left_range = extract_time_range_from_expr(left);
                let right_range = extract_time_range_from_expr(right);
                let mut result = left_range;
                result.intersect(&right_range);
                debug!("Intersected ranges from AND: {:?}", result);
                return result;
            }

            // For OR and other operations, recursively check both sides
            // but we can't easily combine them, so return unbounded
            debug!("Cannot extract time range from operator: {:?}", op);
            TimeRange::unbounded()
        }
        // For other expression types, return unbounded
        _ => {
            debug!("Cannot extract time range from expression type");
            TimeRange::unbounded()
        }
    }
}

/// Extract time ranges from a list of filter expressions
pub fn extract_time_ranges(filters: &[Expr]) -> TimeRange {
    debug!("Extracting time ranges from {} filter(s)", filters.len());

    if filters.is_empty() {
        debug!("No filters provided, returning unbounded range");
        return TimeRange::unbounded();
    }

    let mut combined_range = TimeRange::unbounded();

    for (i, filter) in filters.iter().enumerate() {
        debug!("Processing filter {}: {:?}", i, filter);
        let range = extract_time_range_from_expr(filter);
        combined_range.intersect(&range);
    }

    debug!("Final combined time range: {:?}", combined_range);
    combined_range
}

/// Convert DataFusion filter expression to VParquet4 SpanFilter
/// Returns None if the expression cannot be pushed down to VParquet4Reader
pub fn expr_to_span_filter(expr: &Expr) -> Option<SpanFilter> {
    match expr {
        Expr::BinaryExpr(binary_expr) => {
            let left = &binary_expr.left;
            let right = &binary_expr.right;
            let op = &binary_expr.op;

            // Handle: name = 'literal'
            if matches!(op, Operator::Eq) {
                if let (Expr::Column(col), Expr::Literal(ScalarValue::Utf8(Some(value)), _)) =
                    (left.as_ref(), right.as_ref())
                {
                    if col.name.eq_ignore_ascii_case("name") {
                        debug!(
                            "Converting filter to SpanFilter::NameEquals: {}",
                            value
                        );
                        return Some(SpanFilter::NameEquals(value.clone()));
                    }
                }

                // Handle reversed: 'literal' = name
                if let (Expr::Literal(ScalarValue::Utf8(Some(value)), _), Expr::Column(col)) =
                    (left.as_ref(), right.as_ref())
                {
                    if col.name.eq_ignore_ascii_case("name") {
                        debug!(
                            "Converting filter to SpanFilter::NameEquals (reversed): {}",
                            value
                        );
                        return Some(SpanFilter::NameEquals(value.clone()));
                    }
                }
            }

            None
        }
        _ => None,
    }
}

/// Extract all pushable filters from a list of expressions
/// Currently returns the first supported filter found
/// Future: combine multiple filters with SpanFilter::And
pub fn extract_span_filters(filters: &[Expr]) -> Option<SpanFilter> {
    debug!(
        "Extracting span filters from {} filter expression(s)",
        filters.len()
    );

    for (i, filter) in filters.iter().enumerate() {
        debug!("Checking filter {}: {:?}", i, filter);
        if let Some(sf) = expr_to_span_filter(filter) {
            debug!("Found pushable filter: {:?}", sf);
            return Some(sf);
        }
    }

    debug!("No pushable filters found");
    None
}

#[cfg(test)]
mod tests {
    use super::*;
    use datafusion::logical_expr::{col, lit, BinaryExpr};

    #[test]
    fn test_time_range_unbounded() {
        let range = TimeRange::unbounded();
        assert!(range.is_unbounded());
        assert_eq!(range.min_nanos, None);
        assert_eq!(range.max_nanos, None);
    }

    #[test]
    fn test_time_range_intersect() {
        let mut range1 = TimeRange {
            min_nanos: Some(100),
            max_nanos: Some(200),
        };
        let range2 = TimeRange {
            min_nanos: Some(150),
            max_nanos: Some(250),
        };
        range1.intersect(&range2);
        assert_eq!(range1.min_nanos, Some(150));
        assert_eq!(range1.max_nanos, Some(200));
    }

    #[test]
    fn test_extract_gt_filter() {
        // StartTimeUnixNano > 1000
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::Gt,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(1000)), None)),
        });

        let range = extract_time_range_from_expr(&expr);
        assert_eq!(range.min_nanos, Some(1001));
        assert_eq!(range.max_nanos, None);
    }

    #[test]
    fn test_extract_gte_filter() {
        // StartTimeUnixNano >= 1000
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::GtEq,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(1000)), None)),
        });

        let range = extract_time_range_from_expr(&expr);
        assert_eq!(range.min_nanos, Some(1000));
        assert_eq!(range.max_nanos, None);
    }

    #[test]
    fn test_extract_lt_filter() {
        // StartTimeUnixNano < 2000
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::Lt,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(2000)), None)),
        });

        let range = extract_time_range_from_expr(&expr);
        assert_eq!(range.min_nanos, None);
        assert_eq!(range.max_nanos, Some(1999));
    }

    #[test]
    fn test_extract_lte_filter() {
        // StartTimeUnixNano <= 2000
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::LtEq,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(2000)), None)),
        });

        let range = extract_time_range_from_expr(&expr);
        assert_eq!(range.min_nanos, None);
        assert_eq!(range.max_nanos, Some(2000));
    }

    #[test]
    fn test_extract_eq_filter() {
        // StartTimeUnixNano = 1500
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::Eq,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(1500)), None)),
        });

        let range = extract_time_range_from_expr(&expr);
        assert_eq!(range.min_nanos, Some(1500));
        assert_eq!(range.max_nanos, Some(1500));
    }

    #[test]
    fn test_extract_reversed_filter() {
        // 1000 < StartTimeUnixNano
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(Expr::Literal(ScalarValue::Int64(Some(1000)), None)),
            op: Operator::Lt,
            right: Box::new(col(START_TIME_COLUMN)),
        });

        let range = extract_time_range_from_expr(&expr);
        assert_eq!(range.min_nanos, Some(1001));
        assert_eq!(range.max_nanos, None);
    }

    #[test]
    fn test_extract_and_filter() {
        // StartTimeUnixNano >= 1000 AND StartTimeUnixNano < 2000
        let left = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::GtEq,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(1000)), None)),
        });
        let right = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::Lt,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(2000)), None)),
        });
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(left),
            op: Operator::And,
            right: Box::new(right),
        });

        let range = extract_time_range_from_expr(&expr);
        assert_eq!(range.min_nanos, Some(1000));
        assert_eq!(range.max_nanos, Some(1999));
    }

    #[test]
    fn test_extract_multiple_filters() {
        // Filter list: [StartTimeUnixNano >= 1000, StartTimeUnixNano < 2000]
        let filter1 = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::GtEq,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(1000)), None)),
        });
        let filter2 = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col(START_TIME_COLUMN)),
            op: Operator::Lt,
            right: Box::new(Expr::Literal(ScalarValue::Int64(Some(2000)), None)),
        });

        let range = extract_time_ranges(&[filter1, filter2]);
        assert_eq!(range.min_nanos, Some(1000));
        assert_eq!(range.max_nanos, Some(1999));
    }

    #[test]
    fn test_extract_no_time_filter() {
        // Filter on different column: Name = 'test'
        let expr = Expr::BinaryExpr(BinaryExpr {
            left: Box::new(col("Name")),
            op: Operator::Eq,
            right: Box::new(lit("test")),
        });

        let range = extract_time_range_from_expr(&expr);
        assert!(range.is_unbounded());
    }

    #[test]
    fn test_overlaps_with_block() {
        let block = DiscoveredBlock {
            path: "test".into(),
            size: 100,
            start_time: "2024-01-01T00:00:00Z".to_string(),
            end_time: "2024-01-01T01:00:00Z".to_string(),
        };

        // Parse the block times to get the nanos for testing
        let block_start = DateTime::parse_from_rfc3339("2024-01-01T00:00:00Z")
            .unwrap()
            .timestamp_nanos_opt()
            .unwrap();
        let block_end = DateTime::parse_from_rfc3339("2024-01-01T01:00:00Z")
            .unwrap()
            .timestamp_nanos_opt()
            .unwrap();

        // Range completely before block - should not overlap
        let range = TimeRange {
            min_nanos: Some(block_start - 10000),
            max_nanos: Some(block_start - 1),
        };
        assert!(!range.overlaps_with_block(&block));

        // Range completely after block - should not overlap
        let range = TimeRange {
            min_nanos: Some(block_end + 1),
            max_nanos: Some(block_end + 10000),
        };
        assert!(!range.overlaps_with_block(&block));

        // Range overlaps start of block - should overlap
        let range = TimeRange {
            min_nanos: Some(block_start - 1000),
            max_nanos: Some(block_start + 1000),
        };
        assert!(range.overlaps_with_block(&block));

        // Range overlaps end of block - should overlap
        let range = TimeRange {
            min_nanos: Some(block_end - 1000),
            max_nanos: Some(block_end + 1000),
        };
        assert!(range.overlaps_with_block(&block));

        // Range completely contains block - should overlap
        let range = TimeRange {
            min_nanos: Some(block_start - 1000),
            max_nanos: Some(block_end + 1000),
        };
        assert!(range.overlaps_with_block(&block));

        // Range completely within block - should overlap
        let range = TimeRange {
            min_nanos: Some(block_start + 1000),
            max_nanos: Some(block_end - 1000),
        };
        assert!(range.overlaps_with_block(&block));

        // Unbounded range - should overlap
        let range = TimeRange::unbounded();
        assert!(range.overlaps_with_block(&block));
    }

    #[test]
    fn test_expr_to_span_filter_name_equals() {
        let expr = col("name").eq(lit("test_span"));
        let filter = expr_to_span_filter(&expr);
        assert!(matches!(
            filter,
            Some(SpanFilter::NameEquals(s)) if s == "test_span"
        ));
    }

    #[test]
    fn test_expr_to_span_filter_name_equals_reversed() {
        let expr = lit("test_span").eq(col("name"));
        let filter = expr_to_span_filter(&expr);
        assert!(matches!(
            filter,
            Some(SpanFilter::NameEquals(s)) if s == "test_span"
        ));
    }

    #[test]
    fn test_expr_to_span_filter_case_insensitive() {
        let expr = col("NAME").eq(lit("test_span"));
        let filter = expr_to_span_filter(&expr);
        assert!(matches!(
            filter,
            Some(SpanFilter::NameEquals(s)) if s == "test_span"
        ));
    }

    #[test]
    fn test_expr_to_span_filter_unsupported_column() {
        let expr = col("status_code").eq(lit(1i64));
        let filter = expr_to_span_filter(&expr);
        assert!(filter.is_none());
    }

    #[test]
    fn test_expr_to_span_filter_unsupported_operator() {
        let expr = col("name").gt(lit("test"));
        let filter = expr_to_span_filter(&expr);
        assert!(filter.is_none());
    }

    #[test]
    fn test_extract_span_filters_single_filter() {
        let filters = vec![col("name").eq(lit("test"))];
        let result = extract_span_filters(&filters);
        assert!(matches!(
            result,
            Some(SpanFilter::NameEquals(s)) if s == "test"
        ));
    }

    #[test]
    fn test_extract_span_filters_multiple_filters() {
        let filters = vec![
            col("status_code").eq(lit(1i64)), // unsupported
            col("name").eq(lit("test")),       // supported
        ];
        let result = extract_span_filters(&filters);
        assert!(matches!(
            result,
            Some(SpanFilter::NameEquals(s)) if s == "test"
        ));
    }

    #[test]
    fn test_extract_span_filters_no_supported_filters() {
        let filters = vec![col("status_code").eq(lit(1i64))];
        let result = extract_span_filters(&filters);
        assert!(result.is_none());
    }

    #[test]
    fn test_extract_span_filters_empty() {
        let filters: Vec<Expr> = vec![];
        let result = extract_span_filters(&filters);
        assert!(result.is_none());
    }
}
