package domain

import "fmt"

type Mode string

const (
	ModeRecall Mode = "recall"
	ModeQuiz   Mode = "quiz"
	ModeTypein Mode = "typein"
)

func ParseMode(s string) (Mode, error) {
	m := Mode(s)
	if !m.Valid() {
		return "", fmt.Errorf("invalid mode %q", s)
	}
	return m, nil
}

func (m Mode) Valid() bool {
	switch m {
	case ModeRecall, ModeQuiz, ModeTypein:
		return true
	default:
		return false
	}
}
