package postgres

import (
	"errors"
	"fmt"

	"github.com/drenk83/teleanki/internal/storage"
	"github.com/jackc/pgx/v5/pgconn"
)

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return storage.ErrAlreadyExists
	}
	return err
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
