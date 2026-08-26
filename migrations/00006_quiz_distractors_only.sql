-- +goose Up
UPDATE cards
SET choices = COALESCE((
    SELECT ARRAY(
        SELECT c
        FROM unnest(choices) AS c
        WHERE lower(c) <> lower(back)
    )
), '{}')
WHERE mode = 'quiz';

-- +goose Down
UPDATE cards
SET choices = ARRAY[back] || choices
WHERE mode = 'quiz'
  AND NOT EXISTS (
      SELECT 1 FROM unnest(choices) AS c WHERE lower(c) = lower(back)
  );
