# syntax=docker/dockerfile:1.7
#
# Multi-stage build for backend API and worker.
#
# Build:
#   docker build -f Dockerfile -t feedsystem-backend:api --target api .
#   docker build -f Dockerfile -t feedsystem-backend:worker --target worker .
#

ARG GO_VERSION=1.24.5
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org

FROM golang:${GO_VERSION} AS deps
ARG GOPROXY
ARG GOSUMDB
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

FROM deps AS source
COPY cmd ./cmd
COPY internal ./internal
COPY configs ./configs

FROM source AS test
RUN go mod download
ENV CGO_ENABLED=1
CMD ["go", "test", "-race", "-count=1", "./..."]

FROM source AS build-source
ENV CGO_ENABLED=0

FROM build-source AS api-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/server

FROM build-source AS worker-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM alpine:3.21 AS base
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -s /sbin/nologin app
WORKDIR /app
COPY --from=source /src/configs ./configs
RUN mkdir -p ./.run/uploads && chown -R app:app /app
USER app

FROM base AS api
COPY --from=api-build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM base AS worker
COPY --from=worker-build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]
