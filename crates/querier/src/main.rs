use std::sync::Arc;
use tonic::{transport::Server, Request, Response, Status};

mod http;

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

#[derive(Debug, Default, Clone)]
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

    #[tracing::instrument(skip(self), fields(remote_addr = ?request.remote_addr(), pages_to_search = request.get_ref().pages_to_search, start_page = request.get_ref().start_page))]
    async fn search_block(
        &self,
        request: Request<SearchBlockRequest>,
    ) -> Result<Response<SearchResponse>, Status> {
        tracing::info!("Processing search_block request");
        let request = request.into_inner();
        if request.pages_to_search != std::u32::MAX || request.start_page != 0 {
            return Err(Status::invalid_argument("Rust querier only supports searching the entire block at once"));
        }

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

    // Create shared querier service
    let querier = Arc::new(QuerierService::default());

    // Configure server addresses
    let grpc_addr: std::net::SocketAddr = "0.0.0.0:3200".parse()?;
    let http_addr: std::net::SocketAddr = "0.0.0.0:3100".parse()?;

    tracing::info!(
        grpc_address = %grpc_addr,
        http_address = %http_addr,
        "Starting Tempo Querier servers"
    );

    // Create HTTP server
    let http_handler = http::HttpHandler::new(querier.clone());
    let http_router = http_handler.router();
    let http_server = axum::serve(
        tokio::net::TcpListener::bind(&http_addr).await?,
        http_router.into_make_service(),
    );

    tracing::info!(address = %http_addr, "HTTP server started");

    // Create gRPC server
    let grpc_querier = (*querier).clone();
    let grpc_server = Server::builder()
        .add_service(QuerierServer::new(grpc_querier))
        .serve(grpc_addr);

    tracing::info!(address = %grpc_addr, "gRPC server started");

    // Run both servers concurrently
    tokio::select! {
        result = http_server => {
            if let Err(e) = result {
                tracing::error!(error = %e, "HTTP server error");
            }
        }
        result = grpc_server => {
            if let Err(e) = result {
                tracing::error!(error = %e, "gRPC server error");
            }
        }
        _ = tokio::signal::ctrl_c() => {
            tracing::info!("Shutdown signal received");
        }
    }

    tracing::info!("Tempo Querier shutting down");
    Ok(())
}
