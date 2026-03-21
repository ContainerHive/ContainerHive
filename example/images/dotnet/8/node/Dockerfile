FROM __hive_parent__

USER 0
ARG NODEJS_VERSION
RUN apt-get update \
    && apt-get install --no-install-recommends -y gnupg \
    && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /etc/apt/keyrings  \
    && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_${NODEJS_VERSION}.x nodistro main" | tee /etc/apt/sources.list.d/nodesource.list \
    && apt-get update \
    && apt-get install --no-install-recommends -y nodejs \
    && rm -rf /var/lib/apt/lists/*
USER 1000
