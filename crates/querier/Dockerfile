# Build stage
FROM rust:1.82-bookworm AS builder

WORKDIR /build

# Copy workspace files
COPY Cargo.toml Cargo.lock ./
COPY crates ./crates/

# Copy proto files
COPY pkg/tempopb ./pkg/tempopb/
COPY opentelemetry-proto ./opentelemetry-proto/
COPY vendor ./vendor/

# Install protobuf compiler
RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

# Build the application
RUN cargo build --release --package tempo-querier

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/target/release/tempo-querier /app/tempo-querier

# Expose gRPC port
EXPOSE 9095

# Run the service
ENTRYPOINT ["/app/tempo-querier"]
