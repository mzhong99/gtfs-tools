#!/usr/bin/env bash
set -euo pipefail

IMAGE="gtfs-ephemeral-dev:latest"
REPO_URL="https://github.com/mzhong99/gtfs-tools.git"
REPO_DIR="gtfs"

GIT_NAME="Matthew Zhong"
GIT_EMAIL="matthewzhong@logmethods.com"

echo "[gtfs-dev] Building ephemeral dev image..."

docker build -t "$IMAGE" - <<'EOF'
FROM golang:1.24.4-bookworm

ARG GOPLS_VERSION=v0.18.1
ARG DELVE_VERSION=v1.24.2
ARG AIR_VERSION=v1.61.7

RUN apt-get update && apt-get install -y \
    git vim tmux make curl jq ripgrep ca-certificates \
    postgresql-client protobuf-compiler gh bash-completion \
    && rm -rf /var/lib/apt/lists/*

RUN go install golang.org/x/tools/gopls@${GOPLS_VERSION} \
    && go install github.com/go-delve/delve/cmd/dlv@${DELVE_VERSION} \
    && go install github.com/air-verse/air@${AIR_VERSION}

RUN curl -o /root/.git-prompt.sh \
  https://raw.githubusercontent.com/git/git/master/contrib/completion/git-prompt.sh

RUN echo 'source /root/.git-prompt.sh' >> /root/.bashrc
RUN echo 'export PS1="\u@\h:\w\$ "' >> /root/.bashrc

WORKDIR /workspace
EOF

echo "[gtfs-dev] Starting disposable container..."

docker run --rm -it \
    --name gtfs-ephemeral-dev \
    "$IMAGE" \
    bash -lc "
    set -euo pipefail

    echo '[gtfs-dev] Login to GitHub...'
    gh auth login
    gh auth setup-git

    echo '[gtfs-dev] Cloning repo...'
    git clone '$REPO_URL' '$REPO_DIR'

    cd '$REPO_DIR'

    git config user.name '$GIT_NAME'
    git config user.email '$GIT_EMAIL'

    echo
    echo '[gtfs-dev] Ready.'
    echo 'Repo: /workspace/$REPO_DIR'
    echo 'Exit shell to delete the container and repo checkout.'
    echo

    exec bash
    "
