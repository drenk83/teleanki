package domain

import "time"

type Review struct {
	CardID       int64
	Easiness     float64
	IntervalDays int
	Repetitions  int
	DueAt        time.Time
	UpdatedAt    time.Time
}
