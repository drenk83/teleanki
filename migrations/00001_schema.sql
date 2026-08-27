-- +goose Up
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    telegram_id     BIGINT NOT NULL UNIQUE,
    username        TEXT NOT NULL DEFAULT '',
    daily_limit     INTEGER NOT NULL DEFAULT 20,
    reviews_today   INTEGER NOT NULL DEFAULT 0,
    reviews_on_date DATE NOT NULL DEFAULT DATE '1970-01-01',
    notify_enabled  BOOLEAN NOT NULL DEFAULT true,
    notify_hour     INTEGER NOT NULL DEFAULT 19,
    notify_on_date  DATE NOT NULL DEFAULT DATE '1970-01-01',
    learn_free      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE decks (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    share_code TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE INDEX decks_user_id_idx ON decks (user_id);

CREATE TABLE cards (
    id          BIGSERIAL PRIMARY KEY,
    deck_id     BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    front       TEXT NOT NULL,
    back        TEXT NOT NULL,
    front_image TEXT NOT NULL DEFAULT '',
    back_image  TEXT NOT NULL DEFAULT '',
    mode        TEXT NOT NULL CHECK (mode IN ('recall', 'quiz', 'typein')),
    choices     TEXT[] NOT NULL DEFAULT '{}',
    reversible  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cards_deck_id_idx ON cards (deck_id);

CREATE TABLE reviews (
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id       BIGINT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    easiness      DOUBLE PRECISION NOT NULL DEFAULT 2.5,
    interval_days INTEGER NOT NULL DEFAULT 0,
    repetitions   INTEGER NOT NULL DEFAULT 0,
    due_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, card_id)
);

CREATE INDEX reviews_user_due_idx ON reviews (user_id, due_at);

CREATE TABLE deck_members (
    deck_id    BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (deck_id, user_id)
);

CREATE TABLE user_learn_decks (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, deck_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_learn_decks;
DROP TABLE IF EXISTS deck_members;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS cards;
DROP TABLE IF EXISTS decks;
DROP TABLE IF EXISTS users;
