-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS create_order_responses (
    request_id UUID PRIMARY KEY NOT NULL,
    response JSONB NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS create_order_responses
-- +goose StatementEnd
