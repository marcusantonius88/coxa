-- Criar table outbox
CREATE TABLE IF NOT EXISTS outbox (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    correlation_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP
);

-- Criar table medications (habit-service)
CREATE TABLE IF NOT EXISTS medications (
    id VARCHAR(255) PRIMARY KEY,
    pet_id VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    frequency_days INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Criar table medication_schedules (scheduler-service)
CREATE TABLE IF NOT EXISTS medication_schedules (
    id VARCHAR(255) PRIMARY KEY,
    medication_id VARCHAR(255) NOT NULL,
    pet_id VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    scheduled_date TIMESTAMP NOT NULL,
    next_due_date TIMESTAMP NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, due, overdue, given
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (medication_id) REFERENCES medications(id)
);

-- Criar table medication_administrations (analytics-service)
CREATE TABLE IF NOT EXISTS medication_administrations (
    id VARCHAR(255) PRIMARY KEY,
    medication_id VARCHAR(255) NOT NULL,
    schedule_id VARCHAR(255),
    pet_id VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    admin_date TIMESTAMP NOT NULL,
    notes TEXT,
    next_due_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (medication_id) REFERENCES medications(id)
);

-- Criar table notifications (notification-service)
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(255) PRIMARY KEY,
    medication_id VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    channel VARCHAR(50) DEFAULT 'log', -- log, email, sms, push
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, failed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP,
    FOREIGN KEY (medication_id) REFERENCES medications(id)
);

-- Índices para performance
CREATE INDEX idx_outbox_published ON outbox(published_at);
CREATE INDEX idx_outbox_created ON outbox(created_at);
CREATE INDEX idx_medications_pet ON medications(pet_id);
CREATE INDEX idx_medications_owner ON medications(owner_id);
CREATE INDEX idx_schedules_due ON medication_schedules(next_due_date);
CREATE INDEX idx_schedules_status ON medication_schedules(status);
CREATE INDEX idx_notifications_owner ON notifications(owner_id);
CREATE INDEX idx_notifications_status ON notifications(status);
