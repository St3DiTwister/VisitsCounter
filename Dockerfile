FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/visitsCounter

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/server /app/server

ENV APP_PORT=8080

EXPOSE 8080

ENTRYPOINT ["/app/server"]
