FROM docker.io/library/rust:1.98.1-bookworm@sha256:93ce27a88655056a51dbdd8f5f2d7ddc071c7b0070fb288a37b5a285fc83971e AS build

WORKDIR /build
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release --locked --bin lazyreno

FROM docker.io/library/debian:bookworm-slim@sha256:3783cc01769c7b2b1b83a5c5ad96c815348e28ed7da68e2e3687004faa906251

# native-tls needs OpenSSL and the system certificate store at runtime.
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates libssl3 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 lazyreno \
    && useradd --uid 10001 --gid 10001 --create-home lazyreno

LABEL org.opencontainers.image.title="lazyreno" \
    org.opencontainers.image.description="TUI dashboard for Renovate CE" \
    org.opencontainers.image.licenses="MIT"

COPY --from=build /build/target/release/lazyreno /usr/local/bin/lazyreno

USER 10001:10001
WORKDIR /home/lazyreno
RUN lazyreno --version
ENTRYPOINT ["lazyreno"]
