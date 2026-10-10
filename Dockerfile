FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o touch-grass-quest .

# Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/touch-grass-quest .
COPY --from=builder /app/tasks.json .

# Expose port (default 8080)
EXPOSE 8080

# Run the binary
CMD ["./touch-grass-quest"]
