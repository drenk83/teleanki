package telegram

import (
	"sync"
	"testing"

	"github.com/drenk83/teleanki/internal/domain"
)

func TestSessionStoreImportCardsIsolated(t *testing.T) {
	t.Parallel()
	s := newSessionStore()
	s.set(1, &session{Import: &importDraft{
		Name:  "d",
		Cards: []domain.Card{{Front: "a"}},
	}})
	a := s.get(1)
	b := s.get(1)
	a.Import.Cards[0].Front = "changed"
	if b.Import.Cards[0].Front != "a" {
		t.Fatal("get copies share Import.Cards")
	}
	if s.get(1).Import.Cards[0].Front != "a" {
		t.Fatal("store Import.Cards mutated")
	}
}

func TestSessionStoreConcurrentGetSet(t *testing.T) {
	s := newSessionStore()
	s.set(1, &session{State: stateDeckName, Import: &importDraft{
		Name:  "d",
		Cards: []domain.Card{{Front: "a"}},
	}})
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess := s.get(1)
			if sess.Import != nil && len(sess.Import.Cards) > 0 {
				sess.Import.Cards[0].Front = "x"
			}
			sess.State = stateCardFront
			s.set(1, sess)
			_ = s.get(1)
		}()
	}
	wg.Wait()
}
