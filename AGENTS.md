# AGENTS.md

## О проекте

Telegram-бот интервальных повторений (Anki в Telegram). Модуль: `github.com/drenk83/teleanki`.

UX на русском. Код, идентификаторы, ключи логов и коммиты — на английском.

Бот работает через long polling (`go-telegram/bot`). Вебхуки не использовать.

Ветка разработки — `develop`.

## Команды

```bash
go run ./cmd/bot
go build -o teleanki ./cmd/bot
go test ./...
go vet ./...
gofmt -l .
```

Нужны `TG_TOKEN` и `DATABASE_URL` (env или `.env`). Файл `.env` опционален и в git не коммитится.

Локальная БД: `docker compose up -d` (Postgres 16). DSN в `.env.example` совпадает с `docker-compose.yml`.

## Раскладка

```
cmd/bot/                 вход, wiring, старт
internal/config          env, пользовательские тексты
internal/domain          User, Deck, Card, Review, Mode
internal/scheduler       SM-2
internal/storage         интерфейсы репозиториев
internal/storage/postgres  pgx + goose migrate-on-open
internal/telegram        bot, handlers, middleware, FSM
migrations/              goose SQL, embed.FS
```

`cmd/bot` держать тонким. Хендлеры и middleware — в `internal/telegram`, не в `main.go`.

Пустые `.go` без `package` не оставлять: они ломают `go build ./...` и `go test ./...`.

## Конфиг

- `TG_TOKEN` — обязателен.
- `DATABASE_URL` — обязателен.
- Пользовательские строки — в `internal/config`, не размазывать по хендлерам без нужды.
- Токен не печатать и не логировать.

## Домен

**Mode:** `recall` | `quiz` | `typein`.

**User:** внутренний `ID`, `TelegramID`, `Username`, `DailyLimit` (дефолт 20), счётчик повторений за UTC-день, timestamps.

**Deck:** `Name`, `UserID` (владелец). Имя уникально на пользователя. После trim — 1–64 руны. Режима у колоды нет.

**Card:** `Front`, `Back`, `Mode` (обязателен), `Choices []string`.
Текст front/back после trim — 1–2000 рун.
Для quiz: 2–6 уникальных непустых вариантов, `back` обязан быть среди них. На показе варианты перемешиваются.

**Review:** состояние SM-2 на карточку: `Easiness`, `IntervalDays`, `Repetitions`, `DueAt`.
Новая карточка сразу due (`DueAt = now`, `Easiness = 2.5`, `IntervalDays = 0`, `Repetitions = 0`).

**typein:** `strings.TrimSpace` + `strings.EqualFold`. Нечёткий матч не делать.

## Хранение

PostgreSQL. Handlers ходят только в интерфейсы `internal/storage`, не в `pgx`.

Миграции — goose (уже выбран, не менять). Файлы в `migrations/`, на старте `goose.Up` через embed.

Таблицы: `users`, `decks`, `cards`, `reviews`, `user_learn_decks`.

- `users`: `daily_limit` (1–200, дефолт 20), `reviews_today`, `reviews_on_date`.
- `decks`: FK `user_id` ON DELETE CASCADE, `UNIQUE(user_id, name)`.
- `cards`: FK `deck_id` ON DELETE CASCADE, `mode` NOT NULL CHECK, `choices TEXT[]` NOT NULL DEFAULT `{}`.
- `reviews`: PK `card_id` FK ON DELETE CASCADE.
- `user_learn_decks`: выбор колод для `/learn`. Нет строк — все колоды. FK CASCADE.

Создание карточки и начальный review — в одной транзакции.
Импорт новой колоды (колода + карточки + reviews) — в одной транзакции.

Due-очередь: `ListDue(userID, deckIDs []int64, now, limit)`, сортировка `due_at ASC`.
Пустой `deckIDs` — все колоды пользователя.

Ошибки: `storage.ErrNotFound`, `storage.ErrAlreadyExists`.

## Планировщик SM-2

Пакет `internal/scheduler` не знает про Telegram.

Классический SuperMemo-2. Оценка:

| Кнопка | q |
|--------|---|
| Again  | 1 |
| Hard   | 3 |
| Good   | 4 |
| Easy   | 5 |

```
if q < 3:
    repetitions = 0
    interval = 1
else:
    if repetitions == 0: interval = 1
    elif repetitions == 1: interval = 6
    else: interval = round(interval * EF)
    if interval < 1: interval = 1
    repetitions += 1

EF = EF + (0.1 - (5-q)*(0.08+(5-q)*0.02))
if EF < 1.3: EF = 1.3

due = now + interval days
```

Интервал считается по старому EF, затем EF обновляется (в том числе при провале).
Again не возвращает карточку в ту же сессию (due — завтра).

## Telegram

Только private chat. Группы: короткий отказ, без FSM.

Команды входа:

| Команда    | Действие |
|------------|----------|
| `/start`   | приветствие и объяснения |
| `/menu`    | главное меню |
| `/help`    | справка + схема JSON |
| `/decks`   | список колод |
| `/newdeck` | создать колоду |
| `/learn`   | выбор колод и запуск учёбы |
| документ `.json` | импорт (см. ниже) |

Главное меню (кнопки): Колоды / Учить / Настройки / Помощь.
«В меню» почти на всех экранах, сбрасывает FSM. `/start` ведёт в приветствие, не в меню.

Клики по кнопкам редактируют сообщение бота. Текст или файл пользователя — новое сообщение, старое не удалять.

Внутри сценария — FSM + inline-кнопки. FSM in-memory (`telegram_id → session`), после рестарта бота сессия теряется.

`/help` не сбрасывает FSM. `/start`, `/menu`, `/decks`, `/newdeck`, `/learn` начинают свой сценарий и сбрасывают предыдущий.

Middleware: JSON-лог входящего текста/callback без токена; upsert пользователя. Всегда проверять `From != nil`.

Callback data ≤ 64 байт. Перед действием проверять, что колода/карточка принадлежит пользователю. На callback — `AnswerCallbackQuery`.

Пагинация списков: 5 элементов на страницу.

### CRUD колод

Список: имя → открыть колоду; кнопка «Создать колоду».
Создание: только имя.

Карточка колоды: добавить / список карточек / переименовать / удалить (confirm) / к колодам.

Удаление колоды каскадно удаляет карточки и reviews.

### CRUD карточек

Добавление: front → back → режим (recall/quiz/typein) → если quiz — отвлекающие варианты (по строке, `back` добавляется сам).

Просмотр: front/back/режим/choices; правки полей; удаление с confirm.

### Повторения

- `/learn` — экран выбора колод (сохраняется; пусто = все), затем «Начать».
- Очередь: due из выбранных колод, `due_at ASC`.
- Дневной лимит (1–200, дефолт 20): пресеты 10/20/30/50 или своё число.
- Лимит сессии min(50, остаток на сегодня).
- Пусто → «Нет карточек к повторению». Лимит исчерпан — отдельное сообщение.

| Режим    | Поток                                              | Оценка SM-2              |
|----------|----------------------------------------------------|--------------------------|
| `recall` | вопрос → «Показать» → ответ → Again/Hard/Good/Easy | оценка пользователя      |
| `quiz`   | вопрос + N кнопок (shuffle)                        | верно → Good, нет → Again |
| `typein` | вопрос → пользователь пишет ответ                  | совпало → Good, нет → Again |

После quiz/typein показать верно/неверно (при ошибке — правильный ответ) и сразу следующую карточку.

## Импорт JSON

Только private chat. Документ: имя `*.json` (без учёта регистра) или MIME `application/json`.
Лимит файла 1 MB, карточек ≤ 200. Скачать: `GetFile` → `FileDownloadLink`. JSON — только данные, ничего не исполнять.

Схема (подогнана под домен):

```json
{
  "deck": "Имя",
  "default_mode": "recall",
  "cards": [
    {"front": "...", "back": "...", "mode": "recall"},
    {"front": "...", "back": "...", "mode": "quiz", "choices": ["верный", "нет"]},
    {"front": "...", "back": "..."}
  ]
}
```

- `deck` обязателен. `default_mode` опционален (только для карточек без `mode`, в колоде не хранится).
- `mode` у карточки опционален; иначе берётся `default_mode` или `recall`.
- Для quiz `choices` обязательны и содержат `back`.

Если имя колоды свободно — создать сразу.
Если занято — кнопки: новая «Имя (2)» (при коллизии «Имя (3)»…) / дописать в существующую / отмена.

## Тесты

Обязательны пакеты `internal/domain` и `internal/scheduler` (режимы, effective mode, quiz-инварианты, typein, SM-2: Again, цепочка Good 1→6→EF, min EF 1.3).

## Конвенции

- Пакеты lowercase, идентификаторы EN, строки для пользователя RU.
- Логи: `slog` JSON в stdout, ключи EN.
- Не печатать и не логировать `TG_TOKEN`. `.env` не коммитить.
- Коммиты: короткие lowercase EN, без Conventional Commits, пока так принято в репо.
- Не коммитить, пока явно не попросили.

## Запреты

- Не логировать секреты и не возвращать старый `fmt.Println(tgToken)`.
- Не писать SQL и типы драйвера БД в хендлерах.
- Не смешивать SM-2 с типами Telegram.
- Не добавлять вебхуки, FSRS или нечёткий ввод без запроса.
- Не добавлять зависимости, которых нет в `go.mod`, не проверив модуль.
