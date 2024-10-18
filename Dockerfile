# Stage 1: Build the Go application
FROM golang:1.22.6-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files for dependency management
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o zigbee-watcher .

# Stage 2: Create the final image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/zigbee-watcher .

# Make the binary executable
RUN chmod +x zigbee-watcher

# Command to run the application
CMD ["./zigbee-watcher"]
