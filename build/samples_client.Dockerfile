# syntax=docker/dockerfile:1

FROM golang:1.23.1-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
COPY sdk/ ./sdk/
RUN go mod download

COPY . .
RUN go build -o /bin/s2s-client ./samples/s2s/client-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/s2s-client /s2s-client
ENTRYPOINT ["/s2s-client"]
