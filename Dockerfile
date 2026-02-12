FROM public.ecr.aws/docker/library/golang:1.24-bullseye AS builder

ARG VERSION=dev

WORKDIR /go/src/app

# Copy dependency files first for better caching
COPY go.mod go.sum ./

# Download dependencies (this layer will be cached unless go.mod/go.sum change)
RUN go mod download

# Copy source code (only invalidates cache when source changes)
COPY . .

# Build with CGO disabled for static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-X=main.version=${VERSION}" \
    -o build/aegis ./cmd

FROM public.ecr.aws/docker/library/debian:bullseye-slim

# Install ca-certificates for HTTPS/TLS connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /go/src/app/build/aegis /app/aegis

EXPOSE 8080

ENTRYPOINT ["/app/aegis"]
