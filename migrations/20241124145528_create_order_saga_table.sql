-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION hstore;

ALTER TABLE IF EXISTS orders
ADD COLUMN status varchar(256) NOT NULL DEFAULT '',
ADD COLUMN item_ids_with_stocks hstore DEFAULT NULL,
ADD COLUMN delivery_slot_id BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS items (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    count BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS delivery_slots (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    from_time TIMESTAMP NOT NULL,
    to_time TIMESTAMP NOT NULL,
    deliveries_left BIGINT NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE IF EXISTS orders
DROP COLUMN item_ids_with_stocks,
DROP COLUMN delivery_slot_id;
-- +goose StatementEnd
