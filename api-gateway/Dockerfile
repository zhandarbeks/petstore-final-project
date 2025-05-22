# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy the entire project source code
# This is necessary as the api-gateway imports packages from its own internal directory
# and potentially shared utility packages if you had any at the root, all within the same Go module.
COPY . .

# Build the api-gateway binary
# CGO_ENABLED=0 produces a statically linked binary.
# -o /app/api-gateway-binary specifies the output path and name.
# The build command targets the main.go file of the api-gateway.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/api-gateway-binary ./api-gateway/cmd/main.go

# Stage 2: Create the final, minimal image
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/api-gateway-binary /app/api-gateway-binary

# (Optional) If you have any static assets (like swagger UI files if not embedded)
# or configuration files that need to be bundled, copy them here.
# For example:
# COPY --from=builder /app/api-gateway/docs /app/docs

# Expose the port that the Gin HTTP server will listen on.
# This should match the API_GATEWAY_PORT environment variable.
EXPOSE 8080

# Command to run the executable when the container starts.
CMD ["/app/api-gateway-binary"]
