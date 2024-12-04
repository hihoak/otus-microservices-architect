-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS transactional_outbox_create_order_saga_events_billing_service (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    key varchar(1024) NOT NULL,
    message varchar(4096) NOT NULL
);
CREATE TABLE IF NOT EXISTS transactional_outbox_create_order_saga_events_delivery_service (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    key varchar(1024) NOT NULL,
    message varchar(4096) NOT NULL
);
CREATE TABLE IF NOT EXISTS transactional_outbox_create_order_saga_events_notification_svc (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    key varchar(1024) NOT NULL,
    message varchar(4096) NOT NULL
);
CREATE TABLE IF NOT EXISTS transactional_outbox_create_order_saga_events_warehouse_service (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    key varchar(1024) NOT NULL,
    message varchar(4096) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transactional_outbox_create_order_saga_events_billing_service;
DROP TABLE IF EXISTS transactional_outbox_create_order_saga_events_delivery_service;
DROP TABLE IF EXISTS transactional_outbox_create_order_saga_events_notification_svc;
DROP TABLE IF EXISTS transactional_outbox_create_order_saga_events_warehouse_service;
-- +goose StatementEnd