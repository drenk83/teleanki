-- +goose Up
ALTER TABLE reviews ADD COLUMN user_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
UPDATE reviews r
SET user_id = d.user_id
FROM cards c
JOIN decks d ON d.id = c.deck_id
WHERE c.id = r.card_id;
ALTER TABLE reviews ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE reviews DROP CONSTRAINT reviews_pkey;
ALTER TABLE reviews ADD PRIMARY KEY (user_id, card_id);
CREATE INDEX reviews_user_due_idx ON reviews (user_id, due_at);

ALTER TABLE decks ADD COLUMN share_code TEXT UNIQUE;

CREATE TABLE deck_members (
    deck_id    BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (deck_id, user_id)
);

ALTER TABLE cards ADD COLUMN front_image TEXT NOT NULL DEFAULT '';
ALTER TABLE cards ADD COLUMN back_image TEXT NOT NULL DEFAULT '';

ALTER TABLE users ADD COLUMN notify_enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN notify_hour INTEGER NOT NULL DEFAULT 19;
ALTER TABLE users ADD COLUMN notify_on_date DATE NOT NULL DEFAULT DATE '1970-01-01';

-- +goose Down
ALTER TABLE users DROP COLUMN notify_on_date;
ALTER TABLE users DROP COLUMN notify_hour;
ALTER TABLE users DROP COLUMN notify_enabled;
ALTER TABLE cards DROP COLUMN back_image;
ALTER TABLE cards DROP COLUMN front_image;
DROP TABLE IF EXISTS deck_members;
ALTER TABLE decks DROP COLUMN share_code;
ALTER TABLE reviews DROP CONSTRAINT reviews_pkey;
DROP INDEX IF EXISTS reviews_user_due_idx;
ALTER TABLE reviews DROP COLUMN user_id;
ALTER TABLE reviews ADD PRIMARY KEY (card_id);
