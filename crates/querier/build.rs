fn main() -> Result<(), Box<dyn std::error::Error>> {
    // First compile OpenTelemetry protos
    tonic_build::configure()
        .build_server(false)
        .build_client(false)
        .compile_protos(
            &[
                "../../opentelemetry-proto/opentelemetry/proto/common/v1/common.proto",
                "../../opentelemetry-proto/opentelemetry/proto/resource/v1/resource.proto",
                "../../opentelemetry-proto/opentelemetry/proto/trace/v1/trace.proto",
            ],
            &["../../opentelemetry-proto"],
        )?;

    // Then compile Tempo protos
    tonic_build::configure()
        .build_server(true)
        .build_client(false)
        .compile_protos(
            &["proto/tempo.proto"],
            &[
                "proto",
                "../../opentelemetry-proto",
                "../../vendor",
            ],
        )?;
    Ok(())
}
