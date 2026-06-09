FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor ./vendor
ENV GOFLAGS=-mod=vendor
COPY . .
RUN CGO_ENABLED=0 go build -o /user-service ./cmd/user-service

FROM alpine:3.20
RUN apk add --no-cache ca-certificates netcat-openbsd
COPY --from=builder /user-service /usr/local/bin/user-service
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
