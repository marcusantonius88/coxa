FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY ./shared ./shared
COPY ./services/scheduler-service/go.mod ./
COPY ./services/scheduler-service/main.go ./
RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o scheduler-service main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/scheduler-service .

EXPOSE 8002

CMD ["./scheduler-service"]
