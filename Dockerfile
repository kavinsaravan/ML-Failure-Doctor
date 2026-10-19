# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy backend go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy all backend source code
COPY backend/ ./

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o crashlens .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs python3

WORKDIR /root

# Copy binary from builder
COPY --from=builder /app/crashlens .

# Copy jobs and agents directories
COPY jobs ./jobs
COPY agents ./agents

# Expose port
EXPOSE 8080

# Run the application
CMD ["./crashlens"]
