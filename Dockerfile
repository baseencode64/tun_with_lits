# Multi-stage build for GoXRay VPN Client
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build for Linux
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o goxray .

# Final stage
FROM alpine:3.19

RUN apk add --no-cache \
    iproute2 \
    iputils-ping \
    curl \
    ca-certificates \
    iptables

# Copy binary from builder
COPY --from=builder /app/goxray /usr/local/bin/goxray

# Set capabilities
RUN apk add --no-cache libcap && \
    setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Create TUN device
RUN mkdir -p /dev/net && \
    mknod /dev/net/tun c 10 200 && \
    chmod 600 /dev/net/tun

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ping -c 1 google.com || exit 1

ENTRYPOINT ["goxray"]
CMD ["--help"]
