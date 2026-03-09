# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /audible-plex-downloader ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ffmpeg ca-certificates tzdata

COPY --from=builder /audible-plex-downloader /usr/local/bin/audible-plex-downloader

RUN mkdir -p /config /audiobooks /downloads

EXPOSE 8080

VOLUME ["/config", "/audiobooks", "/downloads"]

ENTRYPOINT ["audible-plex-downloader"]
