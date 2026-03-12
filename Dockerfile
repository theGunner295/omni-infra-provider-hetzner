# Use the official Golang image as a builder
FROM golang:1.26 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build ./cmd/omni-infra-provider-hetzner/

# Start a new stage from scratch
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder --chmod=755 /app/omni-infra-provider-hetzner .
COPY --chmod=755 docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# Start the provider with env-derived flags without exposing secrets in the image config.
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
