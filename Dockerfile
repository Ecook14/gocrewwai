# ---------------------------------------------------------------------------
# Crew-GO — Multi-Stage Production Dockerfile
# ---------------------------------------------------------------------------
# Build:   docker build -t crew-go .
# Run:     docker run -p 9090:9090 -e OPENAI_API_KEY=sk-xxx crew-go
# Health:  curl http://localhost:9090/healthz
# Metrics: curl http://localhost:9090/metrics

# ---- Stage 1: Build ----
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /crew-go ./cmd/crew-go

# ---- Stage 2: Runtime ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata && \
    adduser -D -u 1000 crewgo

WORKDIR /app

COPY --from=builder /crew-go /app/crew-go

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R crewgo:crewgo /app

USER crewgo

# Environment defaults
ENV CREW_GO_SERVER_ADDR=":9090" \
    CREW_GO_LOG_FORMAT="json" \
    CREW_GO_LOG_LEVEL="info" \
    CREW_GO_METRICS_ENABLED="true" \
    CREW_GO_MEMORY_BACKEND="sqlite" \
    CREW_GO_MEMORY_DB_PATH="/app/data/crew_memory.db"

EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:9090/healthz || exit 1

ENTRYPOINT ["/app/crew-go"]
