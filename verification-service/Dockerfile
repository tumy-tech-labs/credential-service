# Use the official Golang image as the base image
FROM golang:1.20-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod ./

# Install any dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o verification-service .

# Start a new stage from scratch
FROM alpine:3.18

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=build /app/verification-service .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./verification-service"]
