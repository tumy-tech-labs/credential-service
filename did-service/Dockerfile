FROM golang:1.21-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o did-service

FROM alpine:3.18
WORKDIR /app
COPY --from=build /app/did-service .
EXPOSE 8080
CMD ["./did-service"]
