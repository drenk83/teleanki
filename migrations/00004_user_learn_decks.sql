-- +goose Up
CREATE TABLE user_learn_decks (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, deck_id)
);

-- +goose Down
DROP TABLE user_learn_decks;
