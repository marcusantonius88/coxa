module coxa/notification-service

go 1.22

require (
	github.com/google/uuid v1.3.0
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.17.0
	github.com/redis/go-redis/v9 v9.0.5
	github.com/segmentio/kafka-go v0.4.42
	coxa/shared v0.0.0
)

replace coxa/shared => ./shared
