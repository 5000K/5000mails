FROM golang:latest AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /5000mails .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY --from=builder /5000mails /5000mails

WORKDIR /

EXPOSE 8080
ENTRYPOINT ["/5000mails"]