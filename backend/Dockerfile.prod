# Build stage
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY backend/ ./

# Build the application
RUN go build -o docker-manager .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS if needed (though we're taking to local docker sock)
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/docker-manager .

# Expose the API port
EXPOSE 8080

# Run the binary
CMD ["./docker-manager"]
