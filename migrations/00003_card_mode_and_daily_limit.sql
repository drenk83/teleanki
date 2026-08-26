-- +goose Up
UPDATE cards SET mode = d.default_mode
FROM decks d
WHERE cards.deck_id = d.id AND cards.mode IS NULL;

ALTER TABLE cards ALTER COLUMN mode SET NOT NULL;

ALTER TABLE decks DROP COLUMN default_mode;

ALTER TABLE users
    ADD COLUMN daily_limit INTEGER NOT NULL DEFAULT 20,
    ADD COLUMN reviews_today INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN reviews_on_date DATE NOT NULL DEFAULT DATE '1970-01-01';

-- +goose Down
ALTER TABLE users
    DROP COLUMN daily_limit,
    DROP COLUMN reviews_today,
    DROP COLUMN reviews_on_date;

ALTER TABLE decks
    ADD COLUMN default_mode TEXT NOT NULL DEFAULT 'recall'
        CHECK (default_mode IN ('recall', 'quiz', 'typein'));

ALTER TABLE cards ALTER COLUMN mode DROP NOT NULL;
