package config

import "github.com/drenk83/teleanki/internal/domain"

var StartMessage = `Привет! Я бот интервальных повторений — Anki в Telegram.

Как пользоваться:
• Создай колоду и добавь карточки (вопрос / ответ / режим).
• В «Учить» выбери колоды и жми «Начать». Берутся только карточки, которым пора повторяться (SM-2).
• «Случайно» — все карточки выбранных колод вперемешку, без лимита и без SM-2.
• Новая карточка сразу попадает в очередь.
• Лимит на день ограничивает, сколько карточек ты оценишь за сутки (МСК) в режиме «Начать».

Режимы карточки:
• вспомнить — смотришь ответ и ставишь Снова / Трудно / Хорошо / Легко;
• тест — выбираешь вариант;
• ввод — пишешь ответ (без учёта регистра).
Для вспомнить и ввод можно включить переворот: карточка с вероятностью 50% покажется с любой стороны.
В вопросе и ответе работает разметка (**жирный**, *курсив*) и форматирование из Telegram.

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
Открой /learn или «Учить» в меню. Отметь колоды (одну, несколько или все) — выбор сохраняется. «Начать» берёт карточки, которым пора, только из отмеченных: сначала самые просроченные. «Случайно» выдаёт все карточки выбранных колод в случайном порядке, без дневного лимита и без SM-2; колода кончилась — заново вперемешку. Пустой выбор = все колоды.

Лимит карточек в день (1–200, по умолчанию 20) — в Настройках или на экране «Учить». Своё число можно написать вручную.

Колоды и карточки
Колода — только имя. Режим задаётся у каждой карточки. Добавление: вопрос → ответ → режим; для теста — неправильные варианты с новой строки (верный добавится сам); для вспомнить и ввод — можно включить переворот сторон.

Импорт JSON
Пришли файл в личку. Имя *.json или MIME application/json.

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
default_mode опционален (recall / quiz / typein) и ставится карточкам без mode; иначе recall.
mode у карточки опционален.
reversible опционален (true/false, по умолчанию false); для quiz игнорируется.
Для quiz нужны choices — только неверные ответы (1–5). Ответ в списке — ошибка. Совпадение без учёта регистра.
В вопросе и ответе: **жирный** *курсив* ~~зачёркнутый~~ или разметка из клиента Telegram.
cards не может быть пустым.
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

	AskCardFront      = "Пришли вопрос текстом или фото с подписью."
	AskCardBack       = "Пришли ответ текстом или фото с подписью."
	AskCardMode       = "Режим этой карточки:"
	AskCardReverse    = "Переворачивать стороны? Карточка будет попадаться и как вопрос, и как ответ."
	AskCardChoices    = "Пришли неправильные варианты, каждый с новой строки (1–5 штук). Правильный ответ добавлю сам."
	AskEditFront      = "Пришли новый вопрос текстом или фото с подписью."
	AskEditBack       = "Пришли новый ответ текстом или фото с подписью."
	AskEditChoices    = "Пришли неправильные варианты, каждый с новой строки (1–5 штук)."
	CardEmptyList     = "В колоде пока нет карточек."
	CardListTitle     = "Карточки колоды «%s»:"
	CardView          = "Вопрос: %s\nОтвет: %s\nРежим: %s%s"
	CardChoicesLine   = "\nНеправильные: %s"
	CardReverseOn     = "\nПереворот: да"
	CardReverseOff    = "\nПереворот: нет"
	ConfirmDeleteCard = "Удалить эту карточку?"
	InvalidCardText   = "Текст: 1–2000 символов."
	InvalidChoices    = "Нужно 1–5 уникальных неправильных вариантов, без пустых строк."

	ReviewEmpty       = "Нет карточек к повторению в выбранных колодах."
	ReviewDone        = "На сегодня всё."
	ReviewBatchDone   = "Пачка закончилась, можно начать ещё."
	ReviewCaughtUp    = "Пока всё — новых карточек нет."
	ReviewProgress    = "Карточка %d из %d\nКолода: %s"
	ReviewRandom      = "Случайно\nКолода: %s"
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

	LearnTitle     = "Учить\nКолоды: %s\nЛимит: %d, осталось сегодня: %d\nОтметь колоды. «Начать» — карточки, которым пора, «Случайно» — все карточки без лимита."
	LearnAllLabel  = "все"
	LearnNoneDecks = "Сначала создай колоду."
	LearnMarkOn    = "✓ "
	LearnMarkOff   = "○ "

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

	BtnOpenMenu        = "Меню"
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
	BtnLearnRandom     = "Случайно"
	BtnLearnNext       = "Дальше"
	BtnReverseOn       = "Переворот: да"
	BtnReverseOff      = "Переворот: нет"
	BtnCustomLimit     = "Своё число"
	BtnNotifyOff       = "Выкл. напоминание"
	BtnNotifyOn        = "Вкл. напоминание"
	BtnNotifyHour      = "Время напоминания"
	BtnShare           = "Поделиться"
	BtnShareRotate     = "Новый код"
	BtnJoin            = "Вступить по коду"
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
