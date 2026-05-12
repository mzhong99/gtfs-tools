#!/usr/bin/env bash
set -euo pipefail

IMAGE="gtfs-ephemeral-dev:latest"
REPO_URL="https://github.com/mzhong99/gtfs-tools.git"
REPO_DIR="gtfs-tools"

GIT_NAME="Matthew Zhong"
GIT_EMAIL="matthewzhong@logmethods.com"

echo "[gtfs-dev] Building ephemeral dev image..."

docker build -t "$IMAGE" - <<'EOF'
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    git \
    gh \
    vim \
    make \
    tmux \
    bash \
    && rm -rf /var/lib/apt/lists/*

RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && chmod a+r /etc/apt/keyrings/docker.gpg

RUN echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/debian \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list

RUN apt-get update && apt-get install -y \
    docker-ce-cli \
    docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
EOF

echo "[gtfs-dev] Starting disposable container..."

docker run --rm -it \
    --name gtfs-ephemeral-dev \
    -v /var/run/docker.sock:/var/run/docker.sock \
    "$IMAGE" \
    bash -lc "
        set -euo pipefail

        gh auth login
        gh auth setup-git

        git clone '$REPO_URL' '$REPO_DIR'
        cd '$REPO_DIR'

        git config user.name '$GIT_NAME'
        git config user.email '$GIT_EMAIL'

        echo
        echo '[gtfs-dev] Ready.'
        echo 'Repo: /workspace/$REPO_DIR'
        echo 'Docker socket mounted.'
        echo 'Exit shell to delete the container and checkout.'
        echo

        exec bash -i
    "
