FROM golang:1.19.0-alpine as builder

WORKDIR /app

COPY mc-ytt-bridge/. .

RUN go mod download

COPY cmd ./cmd
COPY pkg ./pkg

COPY main.go ./main.go

RUN go build -o mc-ytt-bridge main.go

FROM alpine:3.16.0

COPY --from=builder /app/mc-ytt-bridge /app/mc-ytt-bridge

COPY test/handlers /app/handlers

EXPOSE 8080

WORKDIR /app

USER 1001

CMD [ "/app/mc-ytt-bridge", "serve", "--handlers", "/app/handlers" ]
