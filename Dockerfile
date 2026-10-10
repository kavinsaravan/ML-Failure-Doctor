# Build from the repository root: docker build .
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /crashlens .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates sqlite-libs python3 \
    && mkdir -p /app/data
WORKDIR /app
COPY --from=builder /crashlens ./crashlens
COPY jobs/ ./jobs/
ENV PORT=8080 DATABASE_PATH=/app/data/crashlens.db
EXPOSE 8080
CMD ["./crashlens"]
