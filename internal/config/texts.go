package config

import "github.com/drenk83/teleanki/internal/domain"

var StartMessage = `Повторяй карточки в Telegram.

Колоды — свои наборы и чужие, в которые ты вступил. Там создаёшь, правишь, делишься кодом.
Учить — отмечаешь колоды и жмёшь «Начать». Стандартный режим берёт то, чему пора, с лимитом на день. Свободный — все карточки вперемешку, без записи.
Настройки — сколько карточек в день и напоминание.
Помощь — как устроены режимы карточки и как поделиться колодой.`

var HelpMessage = `Как учить
Открой «Учить», отметь колоды, выбери режим и жми «Начать».
Стандартный — карточки, которым пора, с дневным лимитом.
Свободный — все выбранные вперемешку, без записи и лимита.

Карточка
Вопрос, ответ и режим:
• вспомнить — сам вспоминаешь, смотришь ответ, ставишь оценку;
• тест — выбираешь из вариантов;
• ввод — пишешь ответ, регистр не важен.
Переворот (вспомнить и ввод) включается в карточке: стороны меняются местами.
В тексте можно **жирный** *курсив* и форматирование из Telegram. К вопросу и ответу можно фото с подписью.

Поделиться
Открой колоду → «Поделиться» → отправь другу код.
Друг: Колоды → Импортировать → код или файл.

«В меню» сбрасывает то, что ты сейчас вводишь.

Команды: /start /menu /learn /decks /newdeck /help`

const (
	PrivateOnly    = "Я работаю только в личных сообщениях."
	UnknownCommand = "Не понял. Открой /menu или /help."
	UseButtons     = "Выбери вариант кнопкой."
	TryAgain       = "Не получилось, попробуй ещё раз."
	SessionExpired = "Сессия истекла. Начни заново."

	AskDeckName       = "Как назовём колоду? Одно короткое имя, до 64 символов."
	AskDeckRename     = "Новое имя колоды, до 64 символов."
	DeckEmptyList     = "Колод пока нет. Создай свою или импортируй чужую."
	DeckListTitle     = "Колоды · %d–%d из %d"
	DeckView          = "%s\nКарточек: %d"
	ConfirmDeleteDeck = "Удалить колоду «%s» и все карточки?"
	DeckNameTaken     = "Колода с таким именем уже есть."
	InvalidDeckName   = "Имя колоды: 1–64 символа."

	AskCardFront      = "Напиши вопрос — то, что увидишь первым.\nМожно фото: картинка и подпись (текст обязателен)."
	AskCardBack       = "Теперь ответ — то, что должно всплыть в голове.\nСнова: текст или фото с подписью.\nФото на ответе — только режим «вспомнить»."
	AskCardMode       = "Как будешь отвечать на эту карточку?"
	AskCardReverse    = "Показывать карточку и наоборот — то ответ, то вопрос?"
	AskCardChoices    = "Напиши неправильные ответы, каждый с новой строки.\nНужно 1–5 штук, без верного — его подставлю сам."
	AskEditFront      = "Новый вопрос: текст или фото с подписью."
	AskEditBack       = "Новый ответ: текст или фото с подписью."
	AskEditChoices    = "Новые неправильные ответы, каждый с новой строки (1–5, без верного)."
	CardEmptyList     = "В колоде пока нет карточек."
	CardListTitle     = "Карточки колоды «%s»:"
	CardView          = "Вопрос: %s\nОтвет: %s\nРежим: %s%s"
	CardChoicesLine   = "\nНеправильные: %s"
	CardReverseOn     = "\nПереворот: да"
	CardReverseOff    = "\nПереворот: нет"
	ConfirmDeleteCard = "Удалить эту карточку?"
	InvalidCardText   = "Текст: 1–2000 символов."
	InvalidChoices    = "Нужно 1–5 уникальных неправильных вариантов, без пустых строк."

	ReviewEmpty       = "В выбранных колодах сейчас нечего повторять."
	ReviewDone        = "На сегодня лимит исчерпан."
	ReviewBatchDone   = "Пачка кончилась — можно начать ещё."
	ReviewCaughtUp    = "Пока всё, карточек к повторению нет."
	ReviewProgress    = "%d из %d · %s"
	ReviewRandom      = "Свободный · %s"
	ReviewCorrect     = "Верно!"
	ReviewWrong       = "Неверно. Правильно: %s"
	TypeinPrompt      = "Напиши ответ."
	DailyLimitReached = "На сегодня лимит %d карточек. Сменить можно в Настройках."
	SettingsTitle     = "Карточек в день: %d\nОсталось сегодня: %d\nНапоминание: %s"
	SettingsHint      = "Выбери лимит или введи своё число."
	AskCustomLimit    = "Пришли число карточек в день (1–200)."
	InvalidDailyLimit = "Нужно число от 1 до 200."
	NotifyOn          = "вкл, %02d:00 МСК"
	NotifyOff         = "выкл"
	AskNotifyHour     = "В какой час писать? Число 0–23 (МСК)."
	InvalidNotifyHour = "Нужно число от 0 до 23."
	NotifyDue         = "Есть карточки, которым пора. Осталось сегодня: %d."
	PhotoMissing      = "Фото недоступно."
	PhotoNeedCaption  = "Нужен текст в подписи к фото."
	PhotoBadFile      = "Нужно фото jpeg/png/webp до 10 МБ."
	ShareShow         = "Код колоды «%s»:\n%s\nДруг: Колоды → Вступить по коду."
	AskShareCode      = "Пришли код колоды."
	ShareBadCode      = "Нет колоды с таким кодом."
	ShareOwnDeck      = "Это твоя колода."
	ShareJoined       = "Ты в колоде «%s»."
	ShareLeft         = "Ты вышел из колоды."
	ShareMemberView   = "Общая колода: %s\nВладелец: @%s\nКарточек: %d"

	LearnTitle     = "Учить\nКолоды: %s\nЛимит сегодня: %d, осталось: %d\nОтметь колоды. Режим: %s."
	LearnAllLabel  = "все"
	LearnNoneDecks = "Сначала нужна колода — создай или импортируй."
	LearnModeStd   = "стандартный"
	LearnModeFree  = "свободный"
	ImportWait     = `Два способа добавить колоду.

1) Код от друга — 8 символов, например k3mnp7xw. Вставь его сюда.
2) Файл .json (до 1 МБ, до 200 карточек). Пришли документ в этот чат.

Пример файла:
{
  "deck": "Имя колоды",
  "default_mode": "recall",
  "cards": [
    {"front": "вопрос", "back": "ответ"},
    {"front": "hello", "back": "привет", "mode": "typein", "reversible": true},
    {"front": "вопрос", "back": "верный", "mode": "quiz", "choices": ["нет", "может"]}
  ]
}

deck обязателен.
default_mode — для карточек без mode (иначе вспомнить).
mode: recall / quiz / typein.
Для теста choices — только неверные ответы (1–5), без верного.
Пустой cards нельзя.
Если имя колоды занято — новая «Имя (2)» или дописать.

Жду код или файл.`
	LearnMarkOn  = "✓ "
	LearnMarkOff = "○ "

	ImportBadFile        = "Нужен JSON-файл (до 1 МБ, до 200 карточек)."
	ImportBadJSON        = "Не получилось прочитать JSON."
	ImportBadData        = "Файл не подходит: %s"
	ImportEmptyCards     = "В файле нет карточек."
	ImportTooLarge       = "файл слишком большой"
	ImportBadDeckName    = "некорректное имя колоды"
	ImportBadDefaultMode = "некорректный default_mode"
	ImportTooManyCards   = "слишком много карточек"
	ImportBadCardMode    = "карточка %d: некорректный режим"
	ImportBadCardFront   = "карточка %d: некорректный вопрос"
	ImportBadCardBack    = "карточка %d: некорректный ответ"
	ImportBadCardChoices = "карточка %d: некорректные варианты"
	ImportConflict       = "Колода «%s» уже есть. Что сделать?"
	ImportCanceled       = "Импорт отменён."

	BtnMenu            = "В меню"
	BtnMenuDecks       = "Колоды"
	BtnMenuLearn       = "Учить"
	BtnMenuSettings    = "Настройки"
	BtnMenuHelp        = "Помощь"
	BtnDecks           = "К колодам"
	BtnCreateDeck      = "Создать колоду"
	BtnAddCard         = "Добавить"
	BtnCards           = "Карточки"
	BtnRename          = "Переименовать"
	BtnMode            = "Режим"
	BtnDelete          = "Удалить"
	BtnYes             = "Да"
	BtnNo              = "Нет"
	BtnShow            = "Показать"
	BtnAgain           = "Снова"
	BtnHard            = "Трудно"
	BtnGood            = "Хорошо"
	BtnEasy            = "Легко"
	BtnRecall          = "Вспомнить"
	BtnQuiz            = "Тест"
	BtnTypein          = "Ввод"
	BtnEditFront       = "Вопрос"
	BtnEditBack        = "Ответ"
	BtnEditChoices     = "Неправильные ответы"
	BtnBackToDeck      = "К колоде"
	BtnImportNew       = "Новая колода"
	BtnImportAppend    = "Дописать"
	BtnImportCancel    = "Отмена"
	BtnPrev            = "‹"
	BtnNext            = "›"
	BtnLearnAll        = "Все колоды"
	BtnLearnStart      = "Начать"
	BtnLearnModeStd    = "Стандартный"
	BtnLearnModeFree   = "Свободный"
	BtnLearnNext       = "Дальше"
	BtnCancel          = "Отмена"
	BtnImport          = "Импортировать"
	BtnReverseOn       = "Переворот: да"
	BtnReverseOff      = "Переворот: нет"
	BtnCustomLimit     = "Своё число"
	BtnNotifyOff       = "Выкл. напоминание"
	BtnNotifyOn        = "Вкл. напоминание"
	BtnNotifyHour      = "Время напоминания"
	BtnShare           = "Поделиться"
	BtnShareRotate     = "Новый код"
	BtnLeave           = "Выйти"
	BtnClearFrontPhoto = "Убрать фото вопроса"
	BtnClearBackPhoto  = "Убрать фото ответа"
	BtnNotifyLearn     = "Учить"
)

var DailyLimits = []int{10, 20, 30, 50}

func ModeLabel(m domain.Mode) string {
	switch m {
	case domain.ModeRecall:
		return "вспомнить"
	case domain.ModeQuiz:
		return "тест"
	case domain.ModeTypein:
		return "ввод"
	default:
		return string(m)
	}
}
