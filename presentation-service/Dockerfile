# Use the official Golang image as the base image
FROM golang:1.20-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o presentation-service .

FROM alpine:3.18

WORKDIR /

# Copy the Pre-built binary file from the previous stage
COPY --from=build /app/presentation-service /presentation-service

# Command to run the executable
CMD ["/presentation-service"]
