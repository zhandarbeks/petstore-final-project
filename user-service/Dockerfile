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
# This includes all services, proto, genprotos, etc.
# We need this because the user-service might import common packages like genprotos
# from the root of the module.
COPY . .

# Build the user-service binary
# CGO_ENABLED=0 produces a statically linked binary, which is good for alpine.
# -o /app/user-service-binary specifies the output path and name for the compiled application.
# The build command targets the main.go file of the user-service.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/user-service-binary ./user-service/cmd/main.go

# Stage 2: Create the final, minimal image
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/user-service-binary /app/user-service-binary

# (Optional) If you have any static assets or configuration files that need to be
# bundled with the image and are not handled by environment variables, copy them here.
# For example:
# COPY --from=builder /app/user-service/config/some-static-config.json /app/config/some-static-config.json

# Expose the port that the gRPC server will listen on.
# This should match the USER_SERVICE_PORT environment variable.
# The actual port is set via environment variable at runtime,
# but EXPOSE serves as documentation and can be used by some systems.
EXPOSE 50051

# Command to run the executable when the container starts.
# The binary is now at /app/user-service-binary.
CMD ["/app/user-service-binary"]
