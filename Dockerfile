# Use the official Golang image as a builder
FROM golang:1.15 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build ./cmd/omni-infra-provider-hetzner/

# Start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/omni-infra-provider-hetzner .

# Command to run the executable
CMD [ "./omni-infra-provider-hetzner" ]
