// Proto modules
pub mod httpgrpc {
    tonic::include_proto!("httpgrpc");
}

pub mod frontend {
    tonic::include_proto!("frontend");
}

pub mod opentelemetry {
    pub mod proto {
        pub mod common {
            pub mod v1 {
                tonic::include_proto!("opentelemetry.proto.common.v1");
            }
        }
        pub mod resource {
            pub mod v1 {
                tonic::include_proto!("opentelemetry.proto.resource.v1");
            }
        }
        pub mod trace {
            pub mod v1 {
                tonic::include_proto!("opentelemetry.proto.trace.v1");
            }
        }
    }
}

pub mod tempopb {
    tonic::include_proto!("tempopb");
}

// Public modules
pub mod config;
pub mod error;
pub mod http;
pub mod worker;

// Re-exports for convenience
pub use config::WorkerConfig;
pub use error::{QuerierError, Result};
pub use worker::QuerierWorker;
