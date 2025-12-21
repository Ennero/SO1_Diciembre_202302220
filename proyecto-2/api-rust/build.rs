// Le indicamos a Cargo que use tonic_build para compilar los archivos .proto
fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::compile_protos("../proto/ventas.proto")?;
    Ok(())
}