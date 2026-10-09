# Portable AhB Docker image (Linux amd64/arm64). Never build Rust/Go inside the small runtime.
# Debian trixie has new enough glibc for Linux executables compiled on Ubuntu 24.04.
FROM node:22-trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl jq python3 tar coreutils bash \
    && rm -rf /var/lib/apt/lists/*
ARG TARGETARCH
ARG AHB_RELEASE_TAG=linux-0f48b1165b46
RUN set -eux; \
    case "${TARGETARCH:-$(uname -m)}" in amd64|x86_64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) exit 1 ;; esac; \
    case "$AHB_RELEASE_TAG" in linux-????????????) ;; *) exit 1 ;; esac; \
    mkdir -p /opt /tmp/ahb-release; \
    cd /tmp/ahb-release; \
    base="https://github.com/Tsenjii/AhB/releases/download/${AHB_RELEASE_TAG}"; \
    curl -fSL --retry 3 "$base/AhB_linux_${arch}.tar.gz" -o "AhB_linux_${arch}.tar.gz"; \
    curl -fSL --retry 3 "$base/AhB_linux_${arch}.tar.gz.sha256" -o "AhB_linux_${arch}.tar.gz.sha256"; \
    sha256sum -c "AhB_linux_${arch}.tar.gz.sha256"; \
    tar -xzf "AhB_linux_${arch}.tar.gz" -C /opt --no-same-owner; \
    test -x /opt/AhB/bin/hubd; \
    jq -e '.resources.max_running_sidecars == 1 and (.providers | length) == 9' /opt/AhB/config.example.json; \
    rm -rf /tmp/ahb-release
COPY deploy/docker/auth-gateway.mjs /usr/local/bin/ahb-auth-gateway.mjs
COPY deploy/docker/entrypoint.sh /usr/local/bin/ahb-entrypoint
RUN chmod 755 /usr/local/bin/ahb-entrypoint /usr/local/bin/ahb-auth-gateway.mjs \
 && chown -R node:node /opt/AhB \
 && mkdir -p /state && chown node:node /state
USER node
ENV NODE_ENV=production PORT=8080 AHB_STATE_DIR=/state
EXPOSE 8080
WORKDIR /opt/AhB
ENTRYPOINT ["/usr/local/bin/ahb-entrypoint"]
