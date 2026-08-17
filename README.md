# task_tracker

REST API сервис для командной разработки (трекер задач) с ролевым доступом, 
историей изменений, Redis-кэшем и аналитическим SQL отчетом.

## Реализованные функции

**Обязательные требования:**
- [x] API покрывает аутентификацию, команды, задачи, комментарии, историю и аналитику
- [x] Права доступа реализованы на уровне бизнес-логики (подтверждено 33 юнит-тестами)
- [x] Нет утечки данных между командами
- [x] Транзакции для создания/обновления задач и записи истории
- [x] Защита от одновременного обновления задач (optimistic locking)
- [x] Redis кэш для списков задач с корректной инвалидацией
- [x] Swagger/OpenAPI документация
- [x] Понятный README
- [x] Запуск через Docker Compose

**Дополнительные улучшения:**
- [x] Юнит-тесты для бизнес-логики прав доступа
- [x] Ограничение частоты запросов (rate limiting)
- [x] Структурированное логирование
- [ ] Пагинация на основе курсора
- [ ] Circuit breaker (предохранитель)

## Tech stack

- **Language**: Go 1.25
- **HTTP**: [Gin](https://github.com/gin-gonic/gin)
- **Database**: MySQL 8.0
- **Cache**: Redis 7
- **Migrations**: [golang-migrate](https://github.com/golang-migrate/migrate)
- **Auth**: JWT (golang-jwt/jwt/v5 — to be wired in `internal/middleware/auth.go`)
- **Docs**: Swagger / OpenAPI via [swaggo](https://github.com/swaggo/swag)
- **Runtime**: Docker + Docker Compose
- Тестировалось на Ubuntu 24

## Project layout

```
cmd/api/                  Entry point (main.go)
internal/
├── config/               Environment configuration
├── server/               HTTP router + graceful shutdown
├── handler/              HTTP handlers (one file per domain)
├── service/              Business logic (auth, roles, transactions, cache)
├── repository/           Data access: MySQL + Redis
├── model/                Domain types + request/response DTOs
└── middleware/           JWT auth, logger
migrations/               golang-migrate SQL files
docs/swagger/             Generated Swagger assets
docker-compose.yml        Full stack (api + mysql + redis) — for delivery
docker-compose.dev.yml    Infra only (mysql + redis) — for local dev
```

## Быстрый старт (локальная разработка)

Рекомендованный воркфлоу для разработки: запустите MySQL + Redis в Докере, запустите Go app 
локально. 

```bash
# 1. Start infra
make dev-infra

# 2. Run migrations against the local MySQL
make migrate-up

# 3. Run the app
make run
```

API будет доступно на URL: http://localhost:8080. Health check: `GET /health`.

**Swagger UI** (API documentation):
```
http://localhost:8080/swagger/index.html
```

Генерировать swagger docs:
```bash
make swagger
```

## Full stack (Docker)

Для деливери/ревью. Все работает из Docker-контейнеров: API + MySQL + Redis.

### Запуск сервиса в Docker-контейнере 

**Note:** If you changed handler annotations, regenerate swagger docs first:
```bash
make swagger
```
```bash
# 1. Build and start all services (migrations run automatically)
docker compose up --build

# 2. Verify services are running
docker compose ps

# 3. Check API health
curl http://localhost:8080/health
```

API будет доступно на URL: 
```
http://localhost:8080
```
**Swagger UI** (interactive API documentation with testing):
```
http://localhost:8080/swagger/index.html
```

Database migrations run automatically on startup. Check logs to see migration output:
```bash
docker compose logs api
```

### Stopping services

```bash
# Stop containers (data persists in volumes)
docker compose down

# Stop and delete all data (WARNING: loses database)
docker compose down -v
```

### Просмотр логов 

```bash
docker compose logs -f          # all services
docker compose logs -f api      # API only
docker compose logs -f mysql    # MySQL only
```

### Дополнительная информация 

- `api` waits for `mysql` and `redis` to be healthy before starting
- Database data persists across restarts (stored in named volumes)
- Rebuild with `--build` flag after code changes
- Run `make swagger` locally before rebuilding to update API docs

## Migrations

**Full stack (Docker):** Migrations run automatically on container startup.

**Local development:**
```bash
make migrate-up          # apply against local MySQL
make migrate-down        # rollback one step
make migrate-fix         # force schema version (recover from broken state)
```

**Manual Docker migrations** (if needed):
```bash
make docker-migrate-up   # apply inside docker compose
make docker-migrate-down # rollback inside docker compose
```

Файлы миграции находятся в `migrations/`

## Configuration

Весь конфиг считывается из переменных окружения. Смотри в `.env.example` полный список плюс дефолтные значения.

### Конфиг для Local development

Локально, приложение берет `.env` при запуске. Скопируй из файла `.env.example` и поменяй значения, как нужно:

```bash
cp .env.example .env
# Edit .env as needed
```

### Конфиг для Full stack (Docker)

В Docker-контейнере конфигурация задается в `docker-compose.yml` в секции `environment`. Ключевые отличия от локальной разработки:

- `DB_HOST=mysql` (имя сервиса в docker-compose, не `127.0.0.1`)
- `REDIS_HOST=redis` (имя сервиса в docker-compose)
- `APP_ENV=production`

Для изменения значений создайте `.env` файл в корне проекта — docker-compose автоматически прочитает его:

```bash
cp .env.example .env
# Edit .env, then restart:
docker compose down
docker compose up --build
```

### Key variables

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | HTTP port |
| `APP_ENV` | `development` | `development` or `production` (controls Gin mode) |
| `DB_HOST` | `127.0.0.1` | MySQL host (`mysql` in Docker) |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `task_user` | MySQL user |
| `DB_PASS` | `task_pass` | MySQL password |
| `DB_NAME` | `task_tracker` | MySQL database |
| `REDIS_HOST` | `127.0.0.1` | Redis host (`redis` in Docker) |
| `REDIS_PORT` | `6379` | Redis port |
| `JWT_SECRET` | `dev-secret-change-me` | **Change in production** |
| `JWT_EXPIRATION_HOURS` | `24` | Token lifetime |

## Testing

```bash
make test                     # unit + integration + rate limiter tests (все тесты) 
make test-integration         # integration tests (обязательный интеграционный тест) 
make test-access              # unit access rights (опциональный тест прав доступа)  
```

## Enum Values

**Task Status** (`status` field):
- `todo` — not started
- `in_progress` — work in progress
- `done` — completed

**Team Role** (`role` field):
- `owner` — team creator, full access
- `admin` — can invite users and edit any task
- `member` — can create tasks, edit own/assigned tasks

## API Usage Examples

### Authentication

**Register a new user:**
```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "secret123",
    "name": "John Doe"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-08-14T12:00:00Z"
}
```

**Login:**
```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "secret123"
  }'
```

### Teams

**Create a team:**
```bash
curl -X POST http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Development Team"
  }'
```

Response:
```json
{
  "id": 1,
  "name": "Development Team",
  "created_by": 1,
  "created_at": "2026-08-13T12:00:00Z"
}
```

**List your teams:**
```bash
curl -X GET http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Invite a user to a team:**
```bash
curl -X POST http://localhost:8080/api/v1/teams/1/invite \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "jane@example.com",
    "role": "member"
  }'
```

Response:
```json
{
  "team_id": 1,
  "user_id": 2,
  "role": "member"
}
```

### Tasks

**Create a task:**
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "team_id": 1,
    "title": "Implement feature",
    "description": "Add new feature to the system",
    "assignee_id": 2
  }'
```

Response:
```json
{
  "id": 1,
  "team_id": 1,
  "title": "Implement feature",
  "description": "Add new feature to the system",
  "status": "todo",
  "created_by": 1,
  "assignee_id": 2,
  "created_at": "2026-08-13T12:00:00Z",
  "updated_at": "2026-08-13T12:00:00Z",
  "closed_at": null,
  "version": 1
}
```

**List tasks with filters:**
```bash
curl -X GET "http://localhost:8080/api/v1/tasks?team_id=1&status=todo&limit=20&offset=0" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Update a task:**
```bash
curl -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated title",
    "status": "in_progress",
    "version": 1
  }'
```

Response (409 Conflict if version mismatch):
```json
{
  "error": "version mismatch: task was updated by another user"
}
```

**Get task history:**
```bash
curl -X GET http://localhost:8080/api/v1/tasks/1/history \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
[
  {
    "id": 1,
    "task_id": 1,
    "changed_by": 1,
    "changes": {
      "status": {
        "old": "todo",
        "new": "in_progress"
      }
    },
    "created_at": "2026-08-13T12:30:00Z"
  }
]
```

### Comments

**Add a comment:**
```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Working on this task"
  }'
```

**List comments:**
```bash
curl -X GET http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
[
  {
    "id": 1,
    "task_id": 1,
    "user_id": 1,
    "content": "Working on this task",
    "created_at": "2026-08-13T12:30:00Z"
  }
]
```

### Analytics

**Get team statistics:**
```bash
curl -X GET http://localhost:8080/api/v1/teams/1/stats \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "by_status": {
    "todo": 5,
    "in_progress": 3,
    "done": 12
  },
  "top_assignees": [
    {
      "user_id": 2,
      "name": "Alice Johnson",
      "closed_count": 8
    },
    {
      "user_id": 3,
      "name": "Bob Smith",
      "closed_count": 5
    }
  ],
  "avg_close_seconds": 259200.5,
  "total_comments": 47
}
```
