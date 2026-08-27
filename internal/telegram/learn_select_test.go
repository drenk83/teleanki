package telegram

import (
	"testing"

	"github.com/drenk83/teleanki/internal/domain"
)

func TestNextLearnAllSelection(t *testing.T) {
	t.Parallel()
	decks := []domain.Deck{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
		{ID: 3, Name: "c"},
		{ID: 4, Name: "d"},
		{ID: 5, Name: "e"},
		{ID: 6, Name: "f"},
	}
	tests := []struct {
		name     string
		selected []int64
		page     int
		want     []int64
	}{
		{"from all picks first of page 0", nil, 0, []int64{1}},
		{"from all picks first of page 1", nil, 1, []int64{6}},
		{"from subset selects all", []int64{2, 3}, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := nextLearnAllSelection(decks, tt.selected, tt.page)
			if len(got) != len(tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %#v want %#v", got, tt.want)
				}
			}
		})
	}
}
