//! Domain types for vParquet4 traces
//!
//! This module provides OpenTelemetry Protocol (OTLP) types compiled from
//! the official opentelemetry-proto definitions using prost, along with
//! conversion utilities to parse these types from Arrow RecordBatch data.

// Include prost-generated code
pub mod otlp {
    //! OpenTelemetry Protocol (OTLP) types generated from protobuf definitions

    pub mod trace {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/opentelemetry.proto.trace.v1.rs"));
        }
    }

    pub mod common {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/opentelemetry.proto.common.v1.rs"));
        }
    }

    pub mod resource {
        pub mod v1 {
            include!(concat!(env!("OUT_DIR"), "/opentelemetry.proto.resource.v1.rs"));
        }
    }
}

// Conversion utilities for parsing Arrow data into OTLP types
pub mod convert;

// Re-export commonly used types for convenience
pub use otlp::common::v1::{AnyValue, InstrumentationScope, KeyValue};
pub use otlp::resource::v1::Resource;
pub use otlp::trace::v1::{
    span, ResourceSpans, ScopeSpans, Span, Status, TracesData,
};
