# Start with a Golang base image
FROM golang:1.20-alpine

# Set the working directory
WORKDIR /app

# Copy all project files into the container
COPY . .

# Build the Go application
RUN go mod tidy && go build -o verifier-service .

# Expose the service port
EXPOSE 8080

# Start the application
CMD ["./verifier-service"]
