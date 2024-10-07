# Stage 1: Build the application
FROM golang:1.21-alpine AS build

# Set up working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency management
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Run tests during the build stage
# RUN go test ./... -v

# Build the application
RUN go build -o did-service

# Stage 2: Create the final lightweight image
FROM alpine:3.18

# Set up working directory
WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /app/did-service .

# Expose the port
EXPOSE 8080

# Command to run the service
CMD ["./did-service"]
