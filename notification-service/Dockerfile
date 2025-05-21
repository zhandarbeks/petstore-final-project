# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to leverage Docker cache
# These are copied first, so Docker only re-downloads dependencies if these files change.
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy the entire project source code
# This includes all services, proto, genprotos, etc., as they are part of the same Go module.
COPY . .

# Build the notification-service binary
# CGO_ENABLED=0 produces a statically linked binary, good for alpine.
# -o /app/notification-service-binary specifies the output path and name for the compiled application.
# The build command targets the main.go file of the notification-service.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/notification-service-binary ./notification-service/cmd/main.go

# Stage 2: Create the final, minimal image
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/notification-service-binary /app/notification-service-binary

# This service primarily acts as a worker consuming from NATS and making outbound gRPC calls/sending emails.
# It might not expose a port itself unless you add a gRPC server for health checks or other management APIs.
# If you add a gRPC server to notification-service (e.g., in its config.go and main.go), uncomment and set the EXPOSE line.
# EXPOSE 50054

# Command to run the executable when the container starts.
# The binary is now at /app/notification-service-binary.
CMD ["/app/notification-service-binary"]