# ---------------------------------------------------------------------------
# gocrewwai Cloud engine — Dockerfile for Railway / Fly / Render
# Build:   docker build -t gocrew-server .
# Run:     docker run -p 8080:8080 -e OPENAI_API_KEY=sk-xxx gocrew-server
# Health:  curl http://localhost:8080/api/v1/health
# ---------------------------------------------------------------------------

# ---- Stage 1: Build ----
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /gocrew-server ./cmd/server

# ---- Stage 2: Runtime ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata wget && \
    adduser -D -u 1000 gocrew

WORKDIR /app

COPY --from=builder /gocrew-server /app/gocrew-server

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R gocrew:gocrew /app

USER gocrew

# Environment defaults (Railway supplies PORT; we forward as API_PORT)
ENV API_PORT=8080 \
    CREW_GO_LOG_FORMAT="json" \
    CREW_GO_LOG_LEVEL="info" \
    CREW_GO_MEMORY_BACKEND="sqlite" \
    CREW_GO_MEMORY_DB_PATH="/app/data/crew_memory.db"

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget -q --spider http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/gocrew-server"]
