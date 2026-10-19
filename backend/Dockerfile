# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o crashlens .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root

# Copy binary from builder
COPY --from=builder /app/crashlens .

# Create jobs directory for test workloads
RUN mkdir -p ./jobs

# Expose port
EXPOSE 8080

# Run the application
CMD ["./crashlens"]
