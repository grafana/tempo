use std::fs;
use std::io::{Read, Write};
use std::path::PathBuf;

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

    // Read the canonical tempo.proto from pkg/tempopb
    let tempo_proto_path = PathBuf::from("../../pkg/tempopb/tempo.proto");
    let mut tempo_proto_content = String::new();
    fs::File::open(&tempo_proto_path)?.read_to_string(&mut tempo_proto_content)?;

    // Adapt the import paths for Rust build:
    // Change short paths to full OpenTelemetry paths
    tempo_proto_content = tempo_proto_content
        .replace(
            "import \"common/v1/common.proto\"",
            "import \"opentelemetry/proto/common/v1/common.proto\"",
        )
        .replace(
            "import \"trace/v1/trace.proto\"",
            "import \"opentelemetry/proto/trace/v1/trace.proto\"",
        )
        .replace(
            "tempopb.common.v1.KeyValue",
            "opentelemetry.proto.common.v1.KeyValue",
        )
        .replace(
            "tempopb.trace.v1.ResourceSpans",
            "opentelemetry.proto.trace.v1.ResourceSpans",
        )
        .replace(
            "tempopb.trace.v1.Span.Link",
            "opentelemetry.proto.trace.v1.Span.Link",
        );

    // Write adapted proto to a temporary file
    let out_dir = PathBuf::from(std::env::var("OUT_DIR")?);
    let adapted_proto_path = out_dir.join("tempo_adapted.proto");
    fs::File::create(&adapted_proto_path)?.write_all(tempo_proto_content.as_bytes())?;

    // Compile the adapted Tempo proto
    tonic_build::configure()
        .build_server(true)
        .build_client(false)
        .compile_protos(
            &[adapted_proto_path.to_str().unwrap()],
            &[
                out_dir.to_str().unwrap(),
                "../../opentelemetry-proto",
                "../../vendor",
            ],
        )?;

    // Compile httpgrpc and frontend protos for worker functionality
    // Reference the existing proto files directly
    // Compile them together so cross-references work correctly
    tonic_build::configure()
        .build_server(false)
        .build_client(true)
        .compile_protos(
            &[
                "../../vendor/github.com/grafana/dskit/httpgrpc/httpgrpc.proto",
                "../../modules/frontend/v1/frontendv1pb/frontend.proto",
            ],
            &[
                "../../vendor",
                "../../vendor/github.com/gogo/protobuf",
                "../../modules/frontend/v1/frontendv1pb",
            ],
        )?;

    Ok(())
}
