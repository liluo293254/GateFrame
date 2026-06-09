FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /workflow-service ./cmd/workflow-service

FROM alpine:3.20
RUN apk add --no-cache ca-certificates netcat-openbsd
COPY --from=builder /workflow-service /usr/local/bin/workflow-service
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
EXPOSE 8085
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
