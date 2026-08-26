-- +goose Up
ALTER TABLE users ADD COLUMN learn_free BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN learn_free;
