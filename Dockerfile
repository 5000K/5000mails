FROM golang:latest AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /5000mails .

FROM alpine:latest

COPY --from=builder /5000mails /5000mails

WORKDIR /

EXPOSE 8080
ENTRYPOINT ["/5000mails"]