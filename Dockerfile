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
FROM alpine:latest

ENV OMNI_API_ENDPOINT=""
ENV OMNI_SERVICE_ACCOUNT_KEY=""
ENV CONFIG_FILE=""
ENV PROVIDER_NAME=""
ENV ID=""

RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder --chmod=755 /app/omni-infra-provider-hetzner .

# Command to run the executable
CMD /app/omni-infra-provider-hetzner \
    --omni-api-endpoint=$OMNI_API_ENDPOINT \
    --omni-service-account-key=$OMNI_SERVICE_ACCOUNT_KEY \
    --config-file=$CONFIG_FILE \
    --provider-name=$PROVIDER_NAME \
    --id=$ID
