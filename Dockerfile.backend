# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies (gcc, musl-dev for CGO)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY service/ ./service/

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o webapi ./cmd/webapi/

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/webapi .

# Expose port 8080 (as per API spec)
EXPOSE 8080

# Run the application (port 8080 instead of default 3000)
CMD ["./webapi", "-db-filename", "/data/wasatext.db", "--web-api-host", "0.0.0.0:8080"]
