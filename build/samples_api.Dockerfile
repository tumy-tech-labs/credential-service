# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
COPY sdk/ ./sdk/
RUN go mod download

COPY . .
RUN go build -o /bin/s2s-api ./samples/s2s/api-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/s2s-api /s2s-api
EXPOSE 8082
ENTRYPOINT ["/s2s-api"]
