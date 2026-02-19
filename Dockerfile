# syntax=docker/dockerfile:1.7

### Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Build deps
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/document-api ./cmd/api


### Runtime stage
FROM alpine:3.23.3

RUN apk add --no-cache ca-certificates \
  && addgroup -S app \
  && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/document-api /app/document-api

USER app

EXPOSE 8080

ENTRYPOINT ["/app/document-api"]

