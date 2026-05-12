#!/usr/bin/env bash
set -euo pipefail

IMAGE="gtfs-ephemeral-dev:latest"
REPO_URL="https://github.com/mzhong99/gtfs-tools.git"
REPO_DIR="gtfs-tools"

GIT_NAME="Matthew Zhong"
GIT_EMAIL="matthewzhong@logmethods.com"

HOST_WORKDIR="$(mktemp -d "$HOME/gtfs-dev.XXXXXX")"
trap 'sudo rm -rf "$HOST_WORKDIR"' EXIT

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

ARG MIGRATE_VERSION=v4.18.3
RUN curl -L \
    "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.deb" \
    -o /tmp/migrate.deb \
    && apt-get update \
    && apt-get install -y /tmp/migrate.deb \
    && rm -f /tmp/migrate.deb \
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
RUN echo 'export PS1="[dev] \u@\h:\w\\$ "' >> /root/.bashrc
EOF

echo "[gtfs-dev] Starting disposable container..."

CONTAINER_SETUP="$HOST_WORKDIR/container-setup.sh"
cat > "$CONTAINER_SETUP" <<'CONTAINER_SCRIPT'
set -euo pipefail

gh auth login
gh auth setup-git

git clone "$REPO_URL" "$REPO_DIR"
cd "$REPO_DIR"

git config user.name "$GIT_NAME"
git config user.email "$GIT_EMAIL"

cat >> /root/.bashrc <<'BASHRC'
safe_exit() {
    if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        if ! git diff --quiet || ! git diff --cached --quiet; then
            echo
            echo "[gtfs-dev] WARNING: uncommitted git changes detected."
            echo
            git status --short
            echo
            echo "Commit/stash/discard changes before exiting."
            echo "Use 'exit!' to force exit."
            return 1
        fi

        if git rev-parse @{u} >/dev/null 2>&1; then
            LOCAL="$(git rev-parse @)"
            REMOTE="$(git rev-parse @{u})"

            if [ "$LOCAL" != "$REMOTE" ]; then
                echo
                echo "[gtfs-dev] WARNING: unpushed commits detected."
                echo
                git log --oneline @{u}..HEAD
                echo
                echo "Push commits before exiting."
                echo "Use 'exit!' to force exit."
                return 1
            fi
        fi
    fi

    builtin exit
}

force_exit() {
    builtin exit
}

alias exit='safe_exit'
alias logout='safe_exit'
alias exit!='force_exit'
export DB_HOST=host.docker.internal
BASHRC

echo
echo '[gtfs-dev] Ready.'
echo "Repo: /workspace/$REPO_DIR"
echo 'Docker socket mounted.'
echo 'Exit shell to delete the container and checkout.'
echo

exec bash -i
CONTAINER_SCRIPT

docker run --rm -it \
    --name gtfs-ephemeral-dev \
    --add-host=host.docker.internal:host-gateway \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v "$HOME/.vimrc:/root/.vimrc:ro" \
    -v "$HOST_WORKDIR:$HOST_WORKDIR" \
    -w "$HOST_WORKDIR" \
    -e HOST_WORKDIR="$HOST_WORKDIR" \
    -e REPO_URL="$REPO_URL" \
    -e REPO_DIR="$REPO_DIR" \
    -e GIT_NAME="$GIT_NAME" \
    -e GIT_EMAIL="$GIT_EMAIL" \
    "$IMAGE" \
    bash "$HOST_WORKDIR/container-setup.sh"

rm -f "$CONTAINER_SETUP"
echo "[gtfs-dev] Directory cleaned. Done."
