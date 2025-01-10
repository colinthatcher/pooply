# Use the official Golang image as a base image
FROM golang:1.22.5 AS builder

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o main .

# Use a minimal image for the runtime
FROM debian:bullseye-slim

# Install required packages
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose application port
EXPOSE 8080

# Command to run the application
CMD ["./main"]
