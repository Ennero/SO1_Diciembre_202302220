fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Le decimos a tonic que compile nuestro archivo .proto
    // La ruta es: subir un nivel (..) y entrar a proto/ventas.proto
    tonic_build::compile_protos("../proto/ventas.proto")?;
    Ok(())
}