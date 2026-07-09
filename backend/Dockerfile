FROM golang:1.22-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev python3

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o crashlens main.go

EXPOSE 8080

CMD ["./crashlens"]
