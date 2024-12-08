-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS processed_users_events_billing_service (
    id varchar(256) PRIMARY KEY NOT NULL,
    processed_at timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS processed_create_order_saga_commands_billing_service (
    id varchar(256) PRIMARY KEY NOT NULL,
    processed_at timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS processed_create_order_saga_commands_delivery_service (
    id varchar(256) PRIMARY KEY NOT NULL,
    processed_at timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS processed_create_order_saga_commands_notification_service (
    id varchar(256) PRIMARY KEY NOT NULL,
    processed_at timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS processed_create_order_saga_commands_warehouse_service (
    id varchar(256) PRIMARY KEY NOT NULL,
    processed_at timestamp NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS processed_users_events_billing_service;
DROP TABLE IF EXISTS processed_create_order_saga_commands_billing_service;
DROP TABLE IF EXISTS processed_create_order_saga_commands_delivery_service;
DROP TABLE IF EXISTS processed_create_order_saga_commands_notification_service;
DROP TABLE IF EXISTS processed_create_order_saga_commands_warehouse_service;
-- +goose StatementEnd
