# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy source code
COPY main.go .

# Initialize a temporary module for the build
RUN go mod init proxy

# Build the proxy binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /proxy-server main.go

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /proxy-server .

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./proxy-server"]
