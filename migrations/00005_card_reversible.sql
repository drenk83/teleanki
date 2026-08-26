-- +goose Up
ALTER TABLE cards ADD COLUMN reversible BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE cards DROP COLUMN reversible;
