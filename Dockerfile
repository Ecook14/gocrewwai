FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /gocrewwai ./cmd/gocrew
RUN CGO_ENABLED=0 go build -o /gocrewwai-server ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates \
 && addgroup -S gocrew && adduser -S gocrew -G gocrew
COPY --from=builder /gocrewwai /usr/local/bin/gocrewwai
COPY --from=builder /gocrewwai-server /usr/local/bin/gocrewwai-server
USER gocrew

EXPOSE 8080 50051
# Serve the REST API + agent mesh + optional web UI:
#   docker run -p 8080:8080 -e API_AUTH_TOKEN=... gocrewwai --api-port 8080 --web
ENTRYPOINT ["/usr/local/bin/gocrewwai-server"]
CMD ["--api-port", "8080"]
