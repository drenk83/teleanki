package scheduler

import (
	"math"
	"time"

	"github.com/drenk83/teleanki/internal/domain"
)

type Rating int

const (
	RatingAgain Rating = 1
	RatingHard  Rating = 3
	RatingGood  Rating = 4
	RatingEasy  Rating = 5
)

const minEasiness = 1.3

func NewState(now time.Time) domain.Review {
	return domain.Review{
		Easiness:     2.5,
		IntervalDays: 0,
		Repetitions:  0,
		DueAt:        now,
	}
}

func Schedule(state domain.Review, rating Rating, now time.Time) domain.Review {
	q := int(rating)
	if q < 3 {
		state.Repetitions = 0
		state.IntervalDays = 1
	} else {
		switch state.Repetitions {
		case 0:
			state.IntervalDays = 1
		case 1:
			state.IntervalDays = 6
		default:
			state.IntervalDays = int(math.Round(float64(state.IntervalDays) * state.Easiness))
			if state.IntervalDays < 1 {
				state.IntervalDays = 1
			}
		}
		state.Repetitions++
	}

	state.Easiness = state.Easiness + (0.1 - float64(5-q)*(0.08+float64(5-q)*0.02))
	if state.Easiness < minEasiness {
		state.Easiness = minEasiness
	}

	state.DueAt = now.AddDate(0, 0, state.IntervalDays)
	state.UpdatedAt = now
	return state
}
