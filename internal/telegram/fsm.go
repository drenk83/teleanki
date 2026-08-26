package telegram

import (
	"sync"

	"github.com/drenk83/teleanki/internal/domain"
	"github.com/drenk83/teleanki/internal/learn"
)

type state string

const (
	stateIdle           state = ""
	stateDeckName       state = "deck_name"
	stateRenameDeck     state = "rename_deck"
	stateCardFront      state = "card_front"
	stateCardBack       state = "card_back"
	stateCardMode       state = "card_mode"
	stateCardReverse    state = "card_reverse"
	stateCardChoices    state = "card_choices"
	stateEditFront      state = "edit_front"
	stateEditBack       state = "edit_back"
	stateEditChoices    state = "edit_choices"
	stateTypein         state = "typein"
	stateImportConflict state = "import_conflict"
	stateDailyLimit     state = "daily_limit"
)

const (
	pageSize   = 5
	maxImportB = 1 << 20
	maxImportN = 200
)

type session struct {
	State           state
	DeckID          int64
	CardID          int64
	DraftFront      string
	DraftBack       string
	DraftMode       domain.Mode
	DraftReversible bool
	Learn           *learn.Session
	Import          *importDraft
}

type importDraft struct {
	Name       string
	Cards      []domain.Card
	ExistingID int64
}

type sessionStore struct {
	mu sync.Mutex
	m  map[int64]*session
}

func newSessionStore() *sessionStore {
	return &sessionStore{m: make(map[int64]*session)}
}

func (s *sessionStore) get(tgID int64) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.m[tgID]
	if sess == nil {
		return &session{}
	}
	cp := *sess
	if sess.Import != nil {
		imp := *sess.Import
		cp.Import = &imp
	}
	if sess.Learn != nil {
		l := sess.Learn.Clone()
		cp.Learn = &l
	}
	return &cp
}

func (s *sessionStore) set(tgID int64, sess *session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[tgID] = sess
}

func (s *sessionStore) clear(tgID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, tgID)
}
