ARG FEDORA_VERSION=latest

FROM fedora:${FEDORA_VERSION} AS build
WORKDIR /src

RUN dnf -y update && dnf -y install \
    make \
    git \
    golang \
    gcc \
    gcc-c++ \
    ca-certificates \
    && dnf clean all

COPY . .
RUN make

FROM fedora:${FEDORA_VERSION} AS runtime
WORKDIR /app

RUN dnf -y update && \
    dnf -y install migrate ca-certificates && \
    dnf clean all

COPY --from=build /src/bin /app/bin
COPY --from=build /src/config /app/config
COPY --from=build /src/migrations /app/migrations

ENTRYPOINT []
CMD []
