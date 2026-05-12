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
    bash ca-certificates curl docker.io git gh vim tmux make docker-compose \
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
