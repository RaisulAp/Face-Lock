# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=0.1.0
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
      -X github.com/faceclock/faceclock/apps/faceclock-api/internal/version.Version=${VERSION} \
      -X github.com/faceclock/faceclock/apps/faceclock-api/internal/version.Commit=${COMMIT} \
      -X github.com/faceclock/faceclock/apps/faceclock-api/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/faceclock-api ./cmd/api

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates curl && \
    addgroup -S faceclock && adduser -S faceclock -G faceclock && \
    mkdir -p /app/migrations /data/uploads && \
    chown -R faceclock:faceclock /app /data

WORKDIR /app
COPY --from=build /out/faceclock-api /app/faceclock-api
COPY --from=build /src/migrations /app/migrations

USER faceclock
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=5 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/faceclock-api"]
