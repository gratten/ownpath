# Stage 1: Build the Go binary (with CGO for sqlite3)
FROM golang:1.25-alpine AS builder

# Install build dependencies (GCC and tools for CGO)
RUN apk add --no-cache gcc musl-dev git build-base

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with CGO enabled
# -o ownpath: output binary name
# cmd/main.go: your entrypoint
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ownpath ./cmd/main.go

# Stage 2: Create a lightweight runtime image
FROM alpine:latest

# Install runtime dependencies (SQLite for the DB)
RUN apk add --no-cache sqlite

# Set working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/ownpath .

# Copy static web files
COPY --from=builder /app/web ./web

# Expose the port your app listens on (from your Go net/http server)
EXPOSE 8080

# Set a volume for persistent SQLite DB (optional, but good for data persistence in Start9)
VOLUME ["/app/data"]

# Run the binary (assuming your main.go starts the server and uses ./ownpath.db or similar)
CMD ["./ownpath"]