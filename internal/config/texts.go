package config

import "github.com/drenk83/teleanki/internal/domain"

var StartMessage = `Привет. Это бот интервальных повторений.

Свои колоды и чужие, в которые вступил.
Учить — отмечаешь колоды и жмёшь «Начать».
Настройки — лимит на день и напоминание.

Открой меню, чтобы начать.`

var MenuMessage = "Меню"

var HelpMessage = `Как учить
Открой «Учить», отметь колоды, выбери режим и жми «Начать».
Стандартный — карточки, которым пора, с дневным лимитом.
Свободный — все выбранные вперемешку, без записи и лимита.

Как двигается срок
После ответа бот ставит, когда карточку показать снова.
«Снова» — скоро. «Легко» — не скоро.
Чем стабинее помнишь, тем реже вопрос.

Карточка
Вопрос, ответ и режим:
• обычный — сам вспоминаешь, смотришь ответ, ставишь оценку;
• тест — выбираешь из вариантов;
• ввод — пишешь ответ, регистр не важен.
Переворот (обычный и ввод) включается в карточке: стороны меняются местами.
В тексте можно **жирный** *курсив* и форматирование из Telegram. К вопросу и ответу можно фото с подписью.

Поделиться
Открой колоду → «Поделиться» → отправь другу код.
Друг: Колоды → Импортировать → код или файл.
Схема JSON — в «Импортировать».

«В меню» сбрасывает то, что ты сейчас вводишь.

Команды: /start /menu /learn /decks /newdeck /help

Написать: @DRENK83
Код: https://github.com/drenk83/teleanki`

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
	DeckCreated       = "✅ Колода создана.\n\n"
	DeckView          = "Название: %s\nКарточек: %d"
	ConfirmDeleteDeck = "Удалить колоду «%s» и все карточки?"
	DeckNameTaken     = "Колода с таким именем уже есть."
	InvalidDeckName   = "Имя колоды: 1–64 символа."

	AskCardFront      = "Лицевая сторона\nНапиши вопрос — это то, что увидишь первым.\nМожно просто текст. Можно фото: снимок и подпись (подпись обязательна)."
	AskCardBack       = "Обратная сторона\nНапиши ответ — то, что должно всплыть в голове.\nСнова: текст или фото с подписью.\nФото на ответе можно только в режиме «обычный» (после выбора режима)."
	AskCardMode       = "Режим карточки\n\nОбычный — видишь вопрос, вспоминаешь сам, потом смотришь ответ и ставишь оценку.\nТест — вопрос и кнопки с вариантами.\nВвод — пишешь ответ текстом, регистр не важен."
	AskCardReverse    = "Показывать карточку с двух сторон?\nДа — сторона-вопрос выбирается случайно.\nНет — всегда сначала то, что ты написал как вопрос.\n\nПример: «cat» / «кот» → «кот» / «cat»."
	AskCardChoices    = "Напиши неправильные ответы, каждый с новой строки.\nНужно 1–5 штук, без верного — его подставлю сам."
	AskEditFront      = "Новый вопрос: текст или фото с подписью."
	AskEditBack       = "Новый ответ: текст или фото с подписью."
	AskEditChoices    = "Новые неправильные ответы, каждый с новой строки (1–5, без верного)."
	CardEmptyList     = "В колоде пока нет карточек."
	CardListTitle     = "Карточки колоды «%s»:"
	CardSaved         = "✅ Карточка сохранена.\n\n"
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
	ReviewCorrect     = "✅ Верно!"
	ReviewWrong       = "❌ Неверно. Правильно: %s"
	TypeinPrompt      = "Напиши ответ."
	DailyLimitReached = "На сегодня лимит %d карточек. Сменить можно в Настройках."
	SettingsTitle     = "Настройки\n\nКарточек в день: %d\nОсталось сегодня: %d\nНапоминание: %s"
	SettingsHint      = "Лимит — сколько оценок в стандартном «Начать» за сутки (Москва).\nНапоминание — если есть карточки, которым пора.\n\nВыбери лимит или введи своё число."
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
	ShareMemberView   = "Название: %s\nКарточек: %d\nВладелец: @%s"

	LearnTitle     = "Учить\n\nКолоды: %s\nСегодня: лимит %d, осталось: %d\nРежим: %s\n\nСтандартный — бот показывает то, что пора повторить, и считает дневной лимит.\nСвободный — все карточки выбранных колод вперемешку. Повторение не записывается, лимит не тратится.\n\nОтметь колоды и жми «Начать»."
	LearnAllLabel  = "все"
	LearnNoneDecks = "Сначала нужна колода — создай или импортируй."
	LearnModeStd   = "стандартный"
	LearnModeFree  = "свободный"
	ImportWait     = `Добавить колоду

1) Код от друга — 8 символов, вставь сюда.
2) Файл .json — пришли документ в этот чат.
   До 5 МБ и до 10000 карточек. Картинок в файле нет.

Пример:

` + "```json\n" + `{
  "deck": "Имя колоды",
  "cards": [
    {"front": "cat", "back": "кот"},
    {"front": "hello", "back": "привет", "mode": "typein", "reversible": true},
    {"front": "2+2", "back": "4", "mode": "quiz", "choices": ["3", "5"]}
  ]
}
` + "```" + `

deck обязателен. Нет mode у карточки — обычный.
mode: recall — обычный, quiz — тест, typein — ввод.
Для теста choices — только неверные (1–5).
Если имя занято — новая «Имя (2)» или дописать.`
	LearnMarkOn  = "● "
	LearnMarkOff = "○ "

	ImportBadFile        = "Нужен JSON-файл (до 5 МБ, до 10000 карточек)."
	ImportBadJSON        = "Не получилось прочитать JSON."
	ImportBadData        = "Файл не подходит: %s"
	ImportEmptyCards     = "В файле нет карточек."
	ImportTooLarge       = "файл слишком большой"
	ImportBadDeckName    = "некорректное имя колоды"
	ImportBadDefaultMode = "default_mode больше не поддерживается, укажи mode у карточки"
	ImportTooManyCards   = "слишком много карточек"
	ImportBadCardMode    = "карточка %d: некорректный режим"
	ImportBadCardFront   = "карточка %d: некорректный вопрос"
	ImportBadCardBack    = "карточка %d: некорректный ответ"
	ImportBadCardChoices = "карточка %d: некорректные варианты"
	ImportConflict       = "Колода «%s» уже есть. Что сделать?"
	ImportCanceled       = "Импорт отменён."

	BtnMenu            = "В меню"
	BtnOpenMenu        = "Открыть меню"
	BtnMenuDecks       = "Колоды"
	BtnMenuLearn       = "Учить"
	BtnMenuSettings    = "Настройки"
	BtnMenuHelp        = "Помощь"
	BtnDecks           = "К колодам"
	BtnCreateDeck      = "Создать колоду"
	BtnAddCard         = "Добавить"
	BtnAddAnother      = "Добавить ещё"
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
	BtnRecall          = "Обычный"
	BtnReverse         = "Переворот"
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
		return "обычный"
	case domain.ModeQuiz:
		return "тест"
	case domain.ModeTypein:
		return "ввод"
	default:
		return string(m)
	}
}
