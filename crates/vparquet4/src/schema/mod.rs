//! vParquet4 schema definitions and validation
//!
//! This module provides constants for column paths and schema validation
//! for Tempo's vParquet4 trace format.

pub mod field_paths;
pub mod validation;

pub use field_paths::{SpanKind, StatusCode};
pub use validation::validate_schema;
