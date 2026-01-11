# syntax=docker/dockerfile:1

FROM golang:1.23.1-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
COPY sdk/ ./sdk/
RUN go mod download

COPY . .
RUN go build -o /bin/verifier ./cmd/verifier

FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/verifier /verifier
EXPOSE 8081
ENTRYPOINT ["/verifier"]
