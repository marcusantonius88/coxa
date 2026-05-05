FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY ./shared ./shared
COPY ./services/notification-service/go.mod ./
COPY ./services/notification-service/main.go ./
RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/notification-service .

EXPOSE 8003

CMD ["./notification-service"]
