FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o picalc ./cmd/picalc

# Use a minimal alpine image for the final container
FROM alpine:latest

RUN apk --no-cache add ca-certificates && \
    adduser -D -h /app appuser

WORKDIR /app
COPY --from=builder /app/picalc .

# Set the user to run the application
USER appuser

ENTRYPOINT ["/app/picalc"]
CMD ["--help"]
