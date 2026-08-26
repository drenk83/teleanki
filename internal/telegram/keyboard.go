package telegram

import (
	"fmt"
	"strconv"

	"github.com/drenk83/teleanki/internal/config"
	"github.com/go-telegram/bot/models"
)

func btn(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, CallbackData: data}
}

func kb(rows ...[]models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func row(btns ...models.InlineKeyboardButton) []models.InlineKeyboardButton {
	return btns
}

func mainMenuKeyboard() *models.InlineKeyboardMarkup {
	return kb(
		row(btn(config.BtnMenuDecks, "menu:decks"), btn(config.BtnMenuLearn, "menu:learn")),
		row(btn(config.BtnMenuSettings, "menu:settings"), btn(config.BtnMenuHelp, "menu:help")),
	)
}

func modeKeyboard() *models.InlineKeyboardMarkup {
	return kb(row(
		btn(config.BtnRecall, "m:recall"),
		btn(config.BtnQuiz, "m:quiz"),
		btn(config.BtnTypein, "m:typein"),
	))
}

func cardModeKeyboard(cardID int64) *models.InlineKeyboardMarkup {
	id := strconv.FormatInt(cardID, 10)
	return kb(row(
		btn(config.BtnRecall, "c:setmode:"+id+":recall"),
		btn(config.BtnQuiz, "c:setmode:"+id+":quiz"),
		btn(config.BtnTypein, "c:setmode:"+id+":typein"),
	))
}

func reverseKeyboard() *models.InlineKeyboardMarkup {
	return kb(row(
		btn(config.BtnYes, "v:1"),
		btn(config.BtnNo, "v:0"),
	))
}

func nextKeyboard() *models.InlineKeyboardMarkup {
	return kb(row(btn(config.BtnLearnNext, "r:next")))
}

func ratingKeyboard() *models.InlineKeyboardMarkup {
	return kb(row(
		btn(config.BtnAgain, "r:again"),
		btn(config.BtnHard, "r:hard"),
		btn(config.BtnGood, "r:good"),
		btn(config.BtnEasy, "r:easy"),
	))
}

func settingsKeyboard() *models.InlineKeyboardMarkup {
	r := make([]models.InlineKeyboardButton, 0, len(config.DailyLimits))
	for _, n := range config.DailyLimits {
		r = append(r, btn(strconv.Itoa(n), "s:limit:"+strconv.Itoa(n)))
	}
	return kb(r, row(btn(config.BtnCustomLimit, "s:custom")))
}

func pager(page, pages int, prefix string) []models.InlineKeyboardButton {
	if pages <= 1 {
		return nil
	}
	var r []models.InlineKeyboardButton
	if page > 0 {
		r = append(r, btn(config.BtnPrev, fmt.Sprintf("%s:%d", prefix, page-1)))
	}
	if page+1 < pages {
		r = append(r, btn(config.BtnNext, fmt.Sprintf("%s:%d", prefix, page+1)))
	}
	return r
}

func pageSlice[T any](items []T, page int) (chunk []T, p, pages int) {
	n := len(items)
	pages = (n + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	if n == 0 {
		return nil, page, pages
	}
	start := page * pageSize
	end := start + pageSize
	if end > n {
		end = n
	}
	return items[start:end], page, pages
}
