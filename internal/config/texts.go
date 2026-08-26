package config

import "github.com/drenk83/teleanki/internal/domain"

var StartMessage = `Привет! Я бот интервальных повторений — Anki в Telegram.

Как пользоваться:
• Создай колоду и добавь карточки (вопрос / ответ / режим).
• В «Учить» выбери колоды и жми «Начать». Берутся только карточки, которым пора повторяться (SM-2).
• Новая карточка сразу попадает в очередь.
• Лимит на день ограничивает, сколько карточек ты оценишь за сутки (UTC).

Режимы карточки:
• вспомнить — смотришь ответ и ставишь Снова / Трудно / Хорошо / Легко;
• тест — выбираешь вариант;
• ввод — пишешь ответ (без учёта регистра).

Импорт: пришли в личку файл .json (до 1 МБ, до 200 карточек). Формат — в /help.

Команды: /menu /learn /decks /newdeck /help`

var HelpMessage = `Команды
/start — приветствие
/menu — главное меню
/learn — настройка и запуск учёбы
/decks — колоды и карточки
/newdeck — новая колода
/help — эта справка

Меню: Колоды, Учить, Настройки, Помощь. Кнопка «В меню» сбрасывает текущий ввод.

Учить
Открой /learn или «Учить» в меню. Отметь колоды (одну, несколько или все) — выбор сохраняется. «Начать» берёт due-карточки только из отмеченных: сначала самые просроченные. Пустой выбор = все колоды.

Лимит карточек в день (1–200, по умолчанию 20) — в Настройках или на экране «Учить». Своё число можно написать вручную.

Колоды и карточки
Колода — только имя. Режим задаётся у каждой карточки. Добавление: вопрос → ответ → режим; для теста — неправильные варианты с новой строки (верный добавится сам).

Импорт JSON
Пришли файл в личку. Имя *.json или MIME application/json.

{
  "deck": "Имя колоды",
  "default_mode": "recall",
  "cards": [
    {"front": "вопрос", "back": "ответ"},
    {"front": "вопрос", "back": "верный", "mode": "quiz", "choices": ["верный", "нет"]},
    {"front": "вопрос", "back": "ответ", "mode": "typein"}
  ]
}

deck обязателен.
default_mode опционален (recall / quiz / typein) и ставится карточкам без mode; иначе recall.
mode у карточки опционален.
Для quiz нужны choices (2–6) и среди них обязан быть back.
Если колода с таким именем есть — можно создать «Имя (2)» или дописать карточки.`

const (
	MenuTitle = `Главное меню`

	PrivateOnly    = "Я работаю только в личных сообщениях."
	UnknownCommand = "Не понял. Открой /menu или /help."
	UseButtons     = "Выбери вариант кнопкой."
	TryAgain       = "Не получилось, попробуй ещё раз."
	SessionExpired = "Сессия истекла. Начни заново."

	AskDeckName       = "Как назвать колоду?"
	AskDeckRename     = "Пришли новое имя колоды."
	DeckEmptyList     = "Колод пока нет. Создай первую."
	DeckListTitle     = "Твои колоды:"
	DeckView          = "Колода: %s\nКарточек: %d"
	ConfirmDeleteDeck = "Удалить колоду «%s» и все карточки?"
	DeckNameTaken     = "Колода с таким именем уже есть."
	InvalidDeckName   = "Имя колоды: 1–64 символа."

	AskCardFront      = "Пришли вопрос (лицевая сторона)."
	AskCardBack       = "Пришли ответ (оборот)."
	AskCardMode       = "Режим этой карточки:"
	AskCardChoices    = "Пришли неправильные варианты, каждый с новой строки (1–5 штук). Правильный ответ добавлю сам."
	AskEditFront      = "Пришли новый вопрос."
	AskEditBack       = "Пришли новый ответ."
	AskEditChoices    = "Пришли неправильные варианты, каждый с новой строки (1–5 штук)."
	CardEmptyList     = "В колоде пока нет карточек."
	CardListTitle     = "Карточки колоды «%s»:"
	CardView          = "Вопрос: %s\nОтвет: %s\nРежим: %s%s"
	CardChoicesLine   = "\nВарианты: %s"
	ConfirmDeleteCard = "Удалить эту карточку?"
	InvalidCardText   = "Текст: 1–2000 символов."
	InvalidChoices    = "Нужно 1–5 уникальных неправильных вариантов, без пустых строк."

	ReviewEmpty       = "Нет карточек к повторению в выбранных колодах."
	ReviewDone        = "На сегодня всё."
	ReviewProgress    = "Карточка %d из %d\nКолода: %s"
	ReviewCorrect     = "Верно!"
	ReviewWrong       = "Неверно. Правильно: %s"
	TypeinPrompt      = "Напиши ответ."
	DailyLimitReached = "На сегодня лимит %d карточек. Сменить можно в Настройках."
	SettingsTitle     = "Карточек в день: %d\nОсталось сегодня: %d"
	SettingsHint      = "Выбери лимит или введи своё число."
	AskCustomLimit    = "Пришли число карточек в день (1–200)."
	InvalidDailyLimit = "Нужно число от 1 до 200."

	LearnTitle     = "Учить\nКолоды: %s\nЛимит: %d, осталось сегодня: %d\nОтметь колоды и нажми «Начать»."
	LearnAllLabel  = "все"
	LearnNoneDecks = "Сначала создай колоду."
	LearnMarkOn    = "✓ "
	LearnMarkOff   = "○ "

	ImportBadFile  = "Нужен JSON-файл (до 1 МБ, до 200 карточек)."
	ImportBadJSON  = "Не получилось прочитать JSON."
	ImportBadData  = "Файл не подходит: %s"
	ImportConflict = "Колода «%s» уже есть. Что сделать?"
	ImportCanceled = "Импорт отменён."

	BtnOpenMenu     = "Меню"
	BtnMenu         = "В меню"
	BtnMenuDecks    = "Колоды"
	BtnMenuLearn    = "Учить"
	BtnMenuSettings = "Настройки"
	BtnMenuHelp     = "Помощь"
	BtnDecks        = "К колодам"
	BtnCreateDeck   = "Создать колоду"
	BtnAddCard      = "Добавить"
	BtnCards        = "Карточки"
	BtnRename       = "Переименовать"
	BtnMode         = "Режим"
	BtnDelete       = "Удалить"
	BtnYes          = "Да"
	BtnNo           = "Нет"
	BtnShow         = "Показать"
	BtnAgain        = "Снова"
	BtnHard         = "Трудно"
	BtnGood         = "Хорошо"
	BtnEasy         = "Легко"
	BtnRecall       = "Вспомнить"
	BtnQuiz         = "Тест"
	BtnTypein       = "Ввод"
	BtnEditFront    = "Вопрос"
	BtnEditBack     = "Ответ"
	BtnEditChoices  = "Варианты"
	BtnBackToDeck   = "К колоде"
	BtnImportNew    = "Новая колода"
	BtnImportAppend = "Дописать"
	BtnImportCancel = "Отмена"
	BtnPrev         = "‹"
	BtnNext         = "›"
	BtnLearnAll     = "Все колоды"
	BtnLearnStart   = "Начать"
	BtnCustomLimit  = "Своё число"
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
