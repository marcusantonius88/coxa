FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY ./shared ./shared
COPY ./services/habit-service/go.mod ./
COPY ./services/habit-service/main.go ./
RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o habit-service main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/habit-service .

EXPOSE 8001

CMD ["./habit-service"]
