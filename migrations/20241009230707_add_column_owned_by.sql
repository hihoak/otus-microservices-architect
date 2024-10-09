-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD owned_by_username varchar(40) NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
    DROP COLUMN IF EXISTS owned_by_username;
-- +goose StatementEnd
