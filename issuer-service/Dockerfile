FROM golang:1.21-alpine AS build

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code and base schema JSON file into the container
COPY . .

# Copy the base-schema.json file to a specific location (if needed)
# This assumes base-schema.json is in the root of your project
COPY configs/base-schema.json ./configs/base-schema.json

# Build the Go application
RUN go build -o issuer-service

# Start a new stage from alpine base image
FROM alpine:3.18

# Set the working directory
WORKDIR /app

# Copy the built executable from the previous stage
COPY --from=build /app/issuer-service .

# Copy the base schema JSON file into the final image (optional)
COPY --from=build /app/configs/base-schema.json ./configs/base-schema.json

# Expose the port the service runs on
EXPOSE 8080

# Command to run the executable
CMD ["./issuer-service"]
