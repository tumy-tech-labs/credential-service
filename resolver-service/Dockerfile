# Stage 1: Build the Go binary
FROM golang:1.20-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o resolver-service .

# Stage 2: Create a small image
FROM alpine:3.18

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=build /app/resolver-service .

# Expose the port that the service will run on
EXPOSE 8080

# Command to run the executable
CMD ["./resolver-service"]
