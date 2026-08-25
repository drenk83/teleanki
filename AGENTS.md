# AGENTS.md

## О проекте

Telegram-бот интервальных повторений (Anki в Telegram). Модуль: `github.com/drenk83/teleanki`.

UX на русском. Код, идентификаторы, ключи логов и коммиты — на английском.

Сейчас работают только `/start` и `/help` (long polling). Persistence, колоды, карточки и SRS ещё нет. Вся живая логика в `cmd/bot/main.go`; `internal/domain` и `internal/telegram` — заглушки.

## Команды

```bash
go run ./cmd/bot
go build -o teleanki ./cmd/bot
go test ./...
go vet ./...
gofmt -l .
```

Нужен `TG_TOKEN` (env или `.env`). Файл `.env` опционален и в git не коммитится.

Ветка разработки — `develop`.

## Раскладка

```
cmd/bot/                 вход, wiring, старт
internal/config          env, пользовательские тексты
internal/domain          User, Deck, Card, Review, Mode
internal/scheduler       SM-2
internal/storage         интерфейсы репозиториев + postgres
internal/telegram        bot, handlers, middleware, FSM
```

`cmd/bot` держать тонким. Хендлеры и middleware — в `internal/telegram`, не в `main.go`.

Пустые `.go` без `package` не оставлять: они ломают `go build ./...` и `go test ./...`.

## Домен

- **Deck:** имя, владелец, режим по умолчанию.
- **Card:** front/back, режим (свой или наследство колоды). Для `quiz` — варианты ответа.
- **Режимы:** `recall` | `quiz` | `typein`. Выбор при создании колоды или карточки.
- **Review state:** easiness, interval, repetitions, due (поля SM-2).

## Повторения

Команды для входа (`/start`, `/help`, позже создание и `/review`). Внутри сценария — FSM + inline-кнопки.

| Режим    | Поток                                              | Оценка SM-2              |
|----------|----------------------------------------------------|--------------------------|
| `recall` | вопрос → «Показать» → ответ → Again/Hard/Good/Easy | оценка пользователя      |
| `quiz`   | вопрос + N кнопок                                  | верно → Good, нет → Again |
| `typein` | вопрос → пользователь пишет ответ                  | совпало → Good, нет → Again |

`typein`: сравнение после trim и без учёта регистра. Нечёткий матч не делать, пока не попросили.

Алгоритм SRS — **SM-2**. Планировщик не знает про Telegram.

## Хранение

PostgreSQL. Handlers ходят только в интерфейсы `internal/storage`, не в `pgx`.

Миграции — отдельная папка. Инструмент (goose / golang-migrate) выбрать при первой миграции и дальше не менять без нужды.

Таблицы примерно: `users`, `decks`, `cards`, `reviews`.

## Конвенции

- Пакеты lowercase, идентификаторы EN, строки для пользователя RU.
- Логи: `slog` JSON в stdout, ключи EN.
- Не печатать и не логировать `TG_TOKEN`. `.env` не коммитить.
- Коммиты: короткие lowercase EN, без Conventional Commits, пока так принято в репо.
- Не коммитить, пока явно не попросили.

## Роудмап

Делать по порядку, не перескакивать:

1. Починить пустые `.go`, `go mod tidy`, вынести хендлеры в `internal/telegram`.
2. Каркас postgres + миграции + user.
3. Колоды и карточки (CRUD, выбор режима).
4. SM-2 scheduler + due-очередь.
5. Review FSM: сначала `recall`, потом `quiz`, потом `typein`.
6. `/help` по реальным командам; тесты на `domain` и `scheduler`.
7. Импорт карточек из JSON-файла в личке (после шага 3; схему файла не фиксировать заранее).

## Импорт JSON (TODO)

Идея: агент режет конспект на карточки, пользователь кидает `.json` боту в личку, бот парсит и пишет в БД.

Делать **только после** живых `Deck`/`Card` и режимов. Схему JSON не придумывать заранее — подогнать под уже существующую модель. В импорте сразу все три режима: `recall`, `quiz`, `typein`.

Telegram это умеет: документ приходит как `update.Message.Document` (`FileName`, `MimeType`, `FileSize`, `FileID`). У `go-telegram/bot` нет отдельного `HandlerType` — ловить через `RegisterHandlerMatchFunc` или `WithDefaultHandler`. Скачать: `GetFile` → `FileDownloadLink`. Лимит Bot API — 20 MB.

Сейчас документы молча игнорируются (хендлеры только на текст).

Ограничения, когда дойдём: только private chat; лимит размера и числа карточек; JSON — только данные, ничего не исполнять.

## Запреты

- Не логировать секреты и не возвращать старый `fmt.Println(tgToken)`.
- Не писать SQL и типы драйвера БД в хендлерах.
- Не смешивать SM-2 с типами Telegram.
- Не добавлять вебхуки, FSRS или нечёткий ввод без запроса.
- Не добавлять зависимости, которых нет в `go.mod`, не проверив модуль.
