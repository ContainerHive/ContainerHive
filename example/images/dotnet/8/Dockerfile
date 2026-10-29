FROM __hive__/ubuntu:22.04
USER 0
RUN apt-get update -y \
    && apt-get install -y \
        libc6 \
        libgcc-s1 \
        libicu74 \
        liblttng-ust1 \
        libssl3 \
        libstdc++6 \
        zlib1g \
    && rm -rf /var/lib/apt/lists/*
ARG DOTNET_SDK_VERSION
RUN curl -L https://dot.net/v1/dotnet-install.sh -o /usr/bin/install-dotnet \
    && chmod +x /usr/bin/install-dotnet \
    && install-dotnet --version "${DOTNET_SDK_VERSION}" --install-dir /usr/share/dotnet \
    && ln -s /usr/share/dotnet/dotnet /usr/local/bin/dotnet

RUN --mount=type=secret,id=foo \
    CREDENTIALS_FILE=/run/secrets/foo \
    cat /run/secrets/foo
USER 1000
