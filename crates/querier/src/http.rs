use axum::{response::IntoResponse, routing, Router};
use prometheus::{Encoder, Registry, TextEncoder};

/// Create the HTTP router with metrics endpoint
pub fn create_router() -> Router {
    Router::new().route("/metrics", routing::get(metrics_handler))
}

/// Handler for GET /metrics
/// Returns Prometheus metrics in text format
async fn metrics_handler() -> impl IntoResponse {
    // Create an empty registry for now
    // In the future, this will contain actual querier metrics
    let registry = Registry::new();

    // Encode metrics in Prometheus text format
    let encoder = TextEncoder::new();
    let mut buffer = Vec::new();

    let metric_families = registry.gather();
    if let Err(e) = encoder.encode(&metric_families, &mut buffer) {
        tracing::error!(error = %e, "Failed to encode metrics");
        return (
            axum::http::StatusCode::INTERNAL_SERVER_ERROR,
            "Failed to encode metrics",
        )
            .into_response();
    }

    // Return metrics as plain text
    (
        axum::http::StatusCode::OK,
        [(
            axum::http::header::CONTENT_TYPE,
            "text/plain; version=0.0.4",
        )],
        buffer,
    )
        .into_response()
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::http::StatusCode;
    use tower::ServiceExt;

    #[tokio::test]
    async fn test_metrics_endpoint() {
        let router = create_router();

        let request = axum::http::Request::builder()
            .uri("/metrics")
            .body(axum::body::Body::empty())
            .unwrap();

        let response = router.oneshot(request).await.unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        assert_eq!(
            response.headers().get("content-type").unwrap(),
            "text/plain; version=0.0.4"
        );
    }
}
