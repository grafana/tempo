use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_root = PathBuf::from("../../opentelemetry-proto");

    // Compile the trace proto and its dependencies
    prost_build::Config::new()
        .compile_protos(
            &[
                proto_root.join("opentelemetry/proto/trace/v1/trace.proto"),
                proto_root.join("opentelemetry/proto/common/v1/common.proto"),
                proto_root.join("opentelemetry/proto/resource/v1/resource.proto"),
            ],
            &[proto_root],
        )?;

    // Tell Cargo to rerun this build script if proto files change
    println!("cargo:rerun-if-changed=../../opentelemetry-proto");

    Ok(())
}
