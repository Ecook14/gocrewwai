FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /gocrewwai ./cmd/gocrew
RUN CGO_ENABLED=0 go build -o /gocrewwai-server ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /gocrewwai /usr/local/bin/gocrewwai
COPY --from=builder /gocrewwai-server /usr/local/bin/gocrewwai-server

EXPOSE 8080
ENTRYPOINT ["gocrewwai"]
CMD ["serve"]
