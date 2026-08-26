-- +goose Up
CREATE TABLE decks (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    default_mode TEXT NOT NULL CHECK (default_mode IN ('recall', 'quiz', 'typein')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE INDEX decks_user_id_idx ON decks (user_id);

CREATE TABLE cards (
    id         BIGSERIAL PRIMARY KEY,
    deck_id    BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    front      TEXT NOT NULL,
    back       TEXT NOT NULL,
    mode       TEXT CHECK (mode IS NULL OR mode IN ('recall', 'quiz', 'typein')),
    choices    TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cards_deck_id_idx ON cards (deck_id);

CREATE TABLE reviews (
    card_id       BIGINT PRIMARY KEY REFERENCES cards(id) ON DELETE CASCADE,
    easiness      DOUBLE PRECISION NOT NULL DEFAULT 2.5,
    interval_days INTEGER NOT NULL DEFAULT 0,
    repetitions   INTEGER NOT NULL DEFAULT 0,
    due_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reviews_due_at_idx ON reviews (due_at);

-- +goose Down
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS cards;
DROP TABLE IF EXISTS decks;
