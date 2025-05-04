FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o immich-stack ./cmd/main.go

# Use a smaller image for the final container
FROM alpine:latest

WORKDIR /app

# Install bash for the shell script
RUN apk add --no-cache bash

# Copy the binary from builder
COPY --from=builder /app/immich-stack .


# Create a non-root user
RUN adduser -D -g '' appuser
USER appuser

# Set the entrypoint
ENTRYPOINT ["./immich-stack"]