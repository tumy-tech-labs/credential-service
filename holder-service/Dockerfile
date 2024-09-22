# Use the official Go image as the base image
FROM golang:1.21-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the Go modules and download the dependencies
COPY go.mod ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o holder-service .

# Use a smaller image for the final stage
FROM alpine:3.18

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/holder-service .

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["./holder-service"]
