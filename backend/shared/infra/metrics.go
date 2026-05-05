package infra

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ServiceMetrics agrupa todas as métricas de um serviço
type ServiceMetrics struct {
	EventsProcessedTotal    prometheus.Counter
	EventsFailedTotal       prometheus.Counter
	EventProcessingDuration prometheus.Histogram
}

// NewServiceMetrics cria métricas para um serviço
func NewServiceMetrics(serviceName string) *ServiceMetrics {
	return &ServiceMetrics{
		EventsProcessedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total de eventos processados",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		EventsFailedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_failed_total",
			Help: "Total de eventos que falharam",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		EventProcessingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "event_processing_duration_seconds",
			Help: "Duração do processamento de eventos em segundos",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
			Buckets: prometheus.DefBuckets,
		}),
	}
}

// NotificationMetrics agrupa métricas específicas de notificações
type NotificationMetrics struct {
	NotificationsSentTotal   prometheus.Counter
	NotificationsFailedTotal prometheus.Counter
}

// NewNotificationMetrics cria métricas para notification-service
func NewNotificationMetrics() *NotificationMetrics {
	return &NotificationMetrics{
		NotificationsSentTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "notifications_sent_total",
			Help: "Total de notificações enviadas",
		}),
		NotificationsFailedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "notifications_failed_total",
			Help: "Total de notificações que falharam",
		}),
	}
}
