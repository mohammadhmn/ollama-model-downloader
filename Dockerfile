# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git (required for some go modules)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ollama-model-downloader .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/ollama-model-downloader .

# Create downloads directory
RUN mkdir -p downloaded-models

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./ollama-model-downloader", "-port", "8080"]