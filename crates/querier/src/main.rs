use tonic::{transport::Server, Request, Response, Status};

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

use tempopb::querier_server::{Querier, QuerierServer};
use tempopb::{
    SearchBlockRequest, SearchRequest, SearchResponse, SearchTagValuesRequest,
    SearchTagValuesResponse, SearchTagValuesV2Response, SearchTagsRequest, SearchTagsResponse,
    SearchTagsV2Response, TraceByIdRequest, TraceByIdResponse,
};

#[derive(Debug, Default)]
pub struct QuerierService;

#[tonic::async_trait]
impl Querier for QuerierService {
    #[tracing::instrument(skip(self), fields(remote_addr = ?_request.remote_addr()))]
    async fn find_trace_by_id(
        &self,
        _request: Request<TraceByIdRequest>,
    ) -> Result<Response<TraceByIdResponse>, Status> {
        tracing::info!("Processing find_trace_by_id request");
        todo!("find_trace_by_id not yet implemented")
    }

    #[tracing::instrument(skip(self), fields(remote_addr = ?_request.remote_addr()))]
    async fn search_recent(
        &self,
        _request: Request<SearchRequest>,
    ) -> Result<Response<SearchResponse>, Status> {
        tracing::info!("Processing search_recent request");
        todo!("search_recent not yet implemented")
    }

    #[tracing::instrument(skip(self), fields(remote_addr = ?_request.remote_addr()))]
    async fn search_block(
        &self,
        _request: Request<SearchBlockRequest>,
    ) -> Result<Response<SearchResponse>, Status> {
        tracing::info!("Processing search_block request");
        todo!("search_block not yet implemented")
    }

    #[tracing::instrument(skip(self), fields(remote_addr = ?_request.remote_addr()))]
    async fn search_tags(
        &self,
        _request: Request<SearchTagsRequest>,
    ) -> Result<Response<SearchTagsResponse>, Status> {
        tracing::info!("Processing search_tags request");
        todo!("search_tags not yet implemented")
    }

    #[tracing::instrument(skip(self), fields(remote_addr = ?_request.remote_addr()))]
    async fn search_tags_v2(
        &self,
        _request: Request<SearchTagsRequest>,
    ) -> Result<Response<SearchTagsV2Response>, Status> {
        tracing::info!("Processing search_tags_v2 request");
        todo!("search_tags_v2 not yet implemented")
    }

    #[tracing::instrument(skip(self), fields(remote_addr = ?_request.remote_addr()))]
    async fn search_tag_values(
        &self,
        _request: Request<SearchTagValuesRequest>,
    ) -> Result<Response<SearchTagValuesResponse>, Status> {
        tracing::info!("Processing search_tag_values request");
        todo!("search_tag_values not yet implemented")
    }

    #[tracing::instrument(skip(self), fields(remote_addr = ?_request.remote_addr()))]
    async fn search_tag_values_v2(
        &self,
        _request: Request<SearchTagValuesRequest>,
    ) -> Result<Response<SearchTagValuesV2Response>, Status> {
        tracing::info!("Processing search_tag_values_v2 request");
        todo!("search_tag_values_v2 not yet implemented")
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize structured logging with timestamps and log levels
    tracing_subscriber::fmt()
        .with_target(true)
        .with_thread_ids(true)
        .with_line_number(true)
        .init();

    let addr = "0.0.0.0:9095".parse()?;
    let querier = QuerierService::default();

    tracing::info!(address = %addr, "Starting Tempo Querier gRPC server");

    Server::builder()
        .add_service(QuerierServer::new(querier))
        .serve(addr)
        .await?;

    Ok(())
}
