# chat-message-mgz

Микросервис чатов и сообщений на Go + gRPC.

Технические решения:
- PostgreSQL драйвер: `pgx` (`pgxpool`)
- ID чатов: UUIDv7 (генерируются в приложении)
- `message_id` генерируется как `BIGINT` в рамках конкретного чата (1,2,3...)

## Что уже заложено

- Контракт gRPC в `gRPC/service.proto`
- gRPC сервер в `cmd/main.go`
- Реализованные gRPC handlers в `internal/transport/grpc/chat/server.go`
- Слоистый каркас:
  - `internal/storage/postgres` (инициализация пула)
  - `internal/repository/chat` и `internal/repository/messeg` (SQL-реализация по доменам)
  - `internal/repository/interfaces.go` (интерфейсы)
  - `internal/usecase/chat` (бизнес-слой/use-case через интерфейсы репозиториев)
  - `internal/domain` (домен-модели)

## Методы сервиса (MVP)

- `CreateDirectChat` - создать чат между 2 пользователями (без дубликатов)
- `DeleteChat` - удалить чат
- `CreateMessage` - создать сообщение (статус по умолчанию `sent`)
- `SendMessage` - отправить сообщение в чат (`chat_id`, `sender_user_id`, `text`)
- `UpdateMessageStatus` - обновить статус сообщения
- `GetChatPreview` - получить превью чата (последнее сообщение + время)
- `GetLastMessages` - получить последние сообщения (по умолчанию 50) с cursor-пагинацией через `before_message_id`
- `ListUserChats` - получить список чатов пользователя (`other_user_id`, `unread_count`, last message), по умолчанию 15 чатов
- `MarkChatRead` - массово отметить входящие сообщения в чате как `read`

## Установка зависимостей для генерации

```powershell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Убедись, что `%USERPROFILE%\go\bin` есть в `PATH`.

## Генерация protobuf/gRPC кода

Запускать из корня проекта:

```powershell
protoc --proto_path=. --go_out=. --go_opt=module=gitlab.com/siffka/chat-message-mgz --go-grpc_out=. --go-grpc_opt=module=gitlab.com/siffka/chat-message-mgz gRPC/service.proto
```

После этого сгенерируются файлы:

- `pkg/api/chat/v1/service.pb.go`
- `pkg/api/chat/v1/service_grpc.pb.go`

## SQL схема

Схема хранится в миграциях `migrations/`:

- `001_init.up.sql` — применить всю схему
- `001_init.down.sql` — откатить схему

- `chats`
- `messages`
- `chat_user_state` (персональный read-cursor пользователя в чате)
- enum `message_status` (`sent`, `delivered`, `read`)
- триггеры на `updated_at`, preview чата и валидатор переходов статусов (FSM)

Применение миграции вручную:

```powershell
psql "$env:DATABASE_URL" -f .\migrations\001_init.up.sql
```

Откат:

```powershell
psql "$env:DATABASE_URL" -f .\migrations\001_init.down.sql
```

Важно: в схеме `id` без `DEFAULT`, потому что UUIDv7 генерируется в Go-коде репозитория.
Для `messages.id` используется инкремент внутри чата через `chats.last_message_id`.

## Локальный запуск без Docker

```powershell
go mod tidy
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/chat_message?sslmode=disable"
go run .\cmd
```

По умолчанию сервер слушает `:50051`. Порт можно задать через `GRPC_PORT`.
`DATABASE_URL` обязателен.

Минимум репозиторных интерфейсов разделен на:

- `ChatRepository` — операции чатов и превью
- `MessageRepository` — операции сообщений и read/update статусов

## Запуск через Docker Compose

Поднимаются 3 контейнера:
- `postgres` (БД)
- `migrate` (одноразовый контейнер, применяет `migrations/*.up.sql` и завершается)
- `app` (gRPC сервис)

```powershell
docker compose up --build
```

Остановить:

```powershell
docker compose down
```

Остановить и удалить volume с данными БД:

```powershell
docker compose down -v
```

## Smoke-тест клиента

Для проверки есть минимальный gRPC клиент `cmd/client`.

Пример сценария:

```powershell
# 1) Создать чат
go run ./cmd/client create-chat --user1 11111111-1111-1111-1111-111111111111 --user2 22222222-2222-2222-2222-222222222222

# 2) Отправить сообщение
go run ./cmd/client send-message --chat <chat_id> --sender 11111111-1111-1111-1111-111111111111 --text "привет"

# 3) Список чатов пользователя
go run ./cmd/client list-chats --user 11111111-1111-1111-1111-111111111111

# 4) Сообщения чата
go run ./cmd/client get-messages --chat <chat_id> --limit 50
```

Для `GetChatPreview` клиент автоматически отправляет `x-user-id` и `x-trace-id`.

## Swagger / OpenAPI

Файл документации:
- `docs/swagger.yaml`

Быстро открыть Swagger UI локально:

```powershell
docker run --rm -p 8081:8080 -e SWAGGER_JSON=/foo/swagger.yaml -v "${PWD}/docs:/foo" swaggerapi/swagger-ui
```

После запуска открой:
- [http://localhost:8081](http://localhost:8081)

