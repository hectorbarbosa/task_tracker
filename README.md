# task_tracker

REST API сервис для командной разработки (трекер задач) с ролевым доступом, 
историей изменений, Redis-кэшем и аналитическим SQL отчетом.

See `document.pdf` for the full specification.

## Tech stack

- **Language**: Go 1.23
- **HTTP**: [Gin](https://github.com/gin-gonic/gin)
- **Database**: MySQL 8.0
- **Cache**: Redis 7
- **Migrations**: [golang-migrate](https://github.com/golang-migrate/migrate)
- **Auth**: JWT (golang-jwt/jwt/v5 — to be wired in `internal/middleware/auth.go`)
- **Docs**: Swagger / OpenAPI via [swaggo](https://github.com/swaggo/swag)
- **Runtime**: Docker + Docker Compose

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

## Quick start (local development)

Рекомендованный воркфлоу: запустите MySQL + Redis в Докере, запустите Go app 
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

## Full stack (Docker)

Для финального деливери / ревью:

```bash
docker compose up --build       # or: make docker-up
docker compose down             # or: make docker-down
```
`api` контейнер перед запуском ждет стартовавших `mysql` и `redis`.

## Migrations

```bash
make migrate-up          # apply against local MySQL
make migrate-down        # rollback one step
make migrate-fix         # force schema version (recover from broken state)

make docker-migrate-up   # apply inside docker compose (full stack)
```

Файлы миграции находятся в `migrations/`, использование
[golang-migrate naming](https://github.com/golang-migrate/migrate#migration-files):
`{version}_{name}.up.sql` / `{version}_{name}.down.sql`.

## Swagger

```bash
make swagger             # regenerates docs/swagger/*
```

Handlers are annotated with `godoc` comments — see `internal/handler/*.go`.
After generation, mount the UI at `/swagger/*any` (wiring is marked `TODO`
in `internal/server/server.go`).

## Configuration

Весь конфиг считывается из переменных окружения. Смотри в `.env.example` полный
список плюс дефолтные значения. Локально, приложение берет `.env` при запуске, 
скопируй из файла `.env.example` и поменяй значения, как нужно.

Key variables:

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | HTTP port |
| `APP_ENV` | `development` | `development` or `production` (controls Gin mode) |
| `DB_HOST` | `127.0.0.1` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `task_user` | MySQL user |
| `DB_PASS` | `task_pass` | MySQL password |
| `DB_NAME` | `task_tracker` | MySQL database |
| `REDIS_HOST` | `127.0.0.1` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `JWT_SECRET` | `dev-secret-change-me` | **Change in production** |
| `JWT_EXPIRATION_HOURS` | `24` | Token lifetime |

## Testing

```bash
make test-integration         # unit + integration tests (обязательный интеграционный тест) 
make test-access              # unit access rights (опциональный тест прав доступа)  
```

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

## Swagger UI

API документация доступна: 
```
http://localhost:8080/swagger/index.html
```

Генерировать Swagger-документацию
```bash
make swagger
```
