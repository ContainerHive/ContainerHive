# License stage
FROM scratch AS license
COPY LICENSE /LICENSE
COPY NOTICE /NOTICE

# Binary stage - determine architecture and copy appropriate binary
FROM alpine:3 AS bin
RUN apk add --no-cache zstd
COPY ./dist /binaries
RUN if [[ "$(arch)" == "x86_64" ]]; then \
        architecture="amd64"; \
    else \
        architecture="arm64"; \
    fi; \
    ls /binaries \
    && unzstd /binaries/linux-${architecture}.tar.zst \
    && tar -xvf /binaries/linux-${architecture}.tar \
    && cp ch /bin/ch \
    && chmod +x /bin/ch \
    && chown 1000:1000 /bin/ch

# Final stage - use distroless static base
FROM ubuntu:26.04
ENV DEBIAN_FRONTEND="noninteractive"
RUN apt update \
    && apt-get install -y --upgrade ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Add metadata labels
ARG BUILD_TIME \ 
    BUILD_VERSION \
    BUILD_COMMIT_REF
LABEL org.opencontainers.image.title="ContainerHive" \
      org.opencontainers.image.description="ContainerHive - Swarm it. Build it. Run it. Managing container base and library images has never been easier." \
      org.opencontainers.image.licenses='AGPL-3.0' \
      org.opencontainers.image.vendor="Timo Reymann <mail@timo-reymann.de>" \
      org.opencontainers.image.url="https://github.com/timo-reymann/ContainerHive" \
      org.opencontainers.image.source="https://github.com/timo-reymann/ContainerHive.git" \
      org.opencontainers.image.created=${BUILD_TIME} \
      org.opencontainers.image.version=${BUILD_VERSION} \
      org.opencontainers.image.revision=${BUILD_COMMIT_REF}

# Copy license files
COPY --from=license / /

# Copy the architecture-specific ContainerHive binary
COPY --from=bin /bin/ch /bin/ch

USER 1000

# Set entrypoint
ENTRYPOINT ["/bin/ch"]
