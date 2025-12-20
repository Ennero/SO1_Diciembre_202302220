# Documentación de Despliegue y Dockerización (Rust)

Este documento registra los pasos para compilar la API REST/gRPC en Rust y solucionar los requisitos de características experimentales.

## 1. Solución de Conflictos de Versiones
El compilador fallaba solicitando `edition2024`, una característica inestable de Rust.

**Solución aplicada:**
1. Se editó `Cargo.toml` para forzar la edición estable:
   ```toml
   [package]
   edition = "2021"
   ```

2. Se actualizaron dependencias conflictivas (como home) a versiones estables:
    ```bash
        cargo update -p home --precise 0.5.9
    ``` 

## 2. Estrategia de Dockerfile

El Dockerfile de Rust tiene una particularidad: necesita acceder a la carpeta ../proto que está fuera del contexto de la API.

- Contexto de Build: Se usa la raíz del proyecto (.) para poder copiar el archivo ``.proto``.

- Compilador Protoc: Se instala `protobuf-compiler` dentro de la imagen de construcción para que `tonic-build` pueda generar el código cliente.


## 3. Comandos de Construcción y Publicación

Ejecutar desde la raíz del proyecto ``(proyecto-2/)``.


```bash
docker build -f api-rust/Dockerfile -t <IP-EXTERNA-DE-TU-VM>:5000/api-rust:v1 .
```

### Subir Registry Privado

```bash
docker push <IP-EXTERNA-DE-TU-VM>:5000/api-rust:v1
```


