package telegram

import "github.com/drenk83/teleanki/internal/domain"

type cardDraft struct {
	Front      string
	Back       string
	FrontImage string
	BackImage  string
	Mode       domain.Mode
	Reversible bool
}

func (d cardDraft) setFront(text, image string) (cardDraft, error) {
	front, err := domain.NormalizeCardText(text)
	if err != nil {
		return d, err
	}
	d.Front = front
	if image != "" {
		d.FrontImage = image
	}
	return d, nil
}

func (d cardDraft) setBack(text, image string) (cardDraft, error) {
	back, err := domain.NormalizeCardText(text)
	if err != nil {
		return d, err
	}
	d.Back = back
	if image != "" {
		d.BackImage = image
	}
	return d, nil
}

func (d cardDraft) afterBack() (cardDraft, state) {
	if d.BackImage != "" {
		d.Mode = domain.ModeRecall
		d.Reversible = false
		return d, stateCardReverse
	}
	return d, stateCardMode
}

func (d cardDraft) setMode(mode domain.Mode) (cardDraft, state) {
	d.Mode = mode
	d.Reversible = false
	if mode == domain.ModeQuiz {
		return d, stateCardChoices
	}
	return d, stateCardReverse
}

func (d cardDraft) commitNew(deckID int64, distractors []string) (domain.Card, error) {
	c, err := domain.NewCard(deckID, d.Front, d.Back, d.Mode, distractors, d.Reversible)
	if err != nil {
		return domain.Card{}, err
	}
	if d.FrontImage != "" {
		c = c.WithFrontImage(d.FrontImage)
	}
	if d.BackImage != "" {
		c = c.WithBackImage(d.BackImage)
	}
	return c, nil
}

func applyFrontEdit(c domain.Card, text, image string) (domain.Card, error) {
	next, err := c.WithFront(text)
	if err != nil {
		return domain.Card{}, err
	}
	if image != "" {
		next = next.WithFrontImage(image)
	}
	return next, nil
}

type backEdit struct {
	Card        domain.Card
	NeedChoices bool
	DraftBack   string
}

func applyBackEdit(c domain.Card, text, image string) (backEdit, error) {
	next, err := c.WithBack(text)
	if err != nil {
		back, nerr := domain.NormalizeCardText(text)
		if nerr != nil {
			return backEdit{}, nerr
		}
		return backEdit{NeedChoices: true, DraftBack: back}, nil
	}
	if image != "" {
		next = next.WithBackImage(image)
	}
	return backEdit{Card: next}, nil
}
