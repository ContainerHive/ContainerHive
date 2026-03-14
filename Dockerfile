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
FROM gcr.io/distroless/static-debian12:nonroot

# Add metadata labels
ARG BUILD_TIME
ARG BUILD_VERSION
ARG BUILD_COMMIT_REF
LABEL org.opencontainers.image.title="ContainerHive"
LABEL org.opencontainers.image.description="ContainerHive - Container Image Build System"
LABEL org.opencontainers.image.licenses='AGPL-3.0'
LABEL org.opencontainers.image.vendor="Timo Reymann <mail@timo-reymann.de>"
LABEL org.opencontainers.image.url="https://github.com/timo-reymann/ContainerHive"
LABEL org.opencontainers.image.source="https://github.com/timo-reymann/ContainerHive.git"
LABEL org.opencontainers.image.created=${BUILD_TIME}
LABEL org.opencontainers.image.version=${BUILD_VERSION}
LABEL org.opencontainers.image.revision=${BUILD_COMMIT_REF}

# Copy license files
COPY --from=license / /

# Copy the architecture-specific ContainerHive binary
COPY --from=bin /bin/ch /bin/ch

# Set entrypoint
ENTRYPOINT ["/bin/ch"]
