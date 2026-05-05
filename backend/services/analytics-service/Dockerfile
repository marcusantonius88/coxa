FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY ./shared ./shared
COPY ./services/analytics-service/go.mod ./
COPY ./services/analytics-service/main.go ./
RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o analytics-service main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/analytics-service .

EXPOSE 8004

CMD ["./analytics-service"]
