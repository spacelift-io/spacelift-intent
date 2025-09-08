# Build stage
FROM golang:1.24-alpine@sha256:b4f875e650466fa0fe62c6fd3f02517a392123eea85f1d7e69d85f780e4db1c1 AS builder

# Install build dependencies including protoc and curl
RUN apk add --no-cache git ca-certificates protobuf-dev curl

# Set working directory
WORKDIR /app

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/standalone cmd/standalone
COPY instructions instructions
COPY proto proto
COPY provider provider
COPY registry registry
COPY storage storage
COPY tools tools
COPY types types

# Generate proto files dynamically
RUN go generate ./...

RUN mkdir -p /app/bin

# Build the application (pure Go, no CGO needed)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/standalone ./cmd/standalone


# Runtime stage
FROM alpine:latest@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715

# Install runtime dependencies
RUN apk add --no-cache wget unzip ca-certificates

# Create app directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/bin/standalone .

# Expose stdio for MCP protocol
CMD ["./standalone"]
