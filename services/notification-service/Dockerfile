FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /notification-service ./cmd/notification-service

FROM alpine:3.20
RUN apk add --no-cache ca-certificates netcat-openbsd
COPY --from=builder /notification-service /usr/local/bin/notification-service
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
EXPOSE 8087
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
