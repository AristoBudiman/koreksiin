FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod ./
COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /koreksiin-server ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /koreksiin-server /app/koreksiin-server
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8000

CMD ["/app/koreksiin-server"]
