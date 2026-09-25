-- +goose Up
ALTER TABLE users
    ADD COLUMN full_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users
    DROP COLUMN full_name;
