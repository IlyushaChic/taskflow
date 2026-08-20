# 📌 О проекте

**TaskFlow** — это менеджер задач, который будет расти от простого CRUD-приложения до полноценной распределённой системы с асинхронностью, аналитикой, стримингом, оркестрацией и полным observability-стеком.

---

## 🧱 Сущности

- **Задача**: `id`, `title`, `description`, `status` (`new`, `in_progress`, `done`, `cancelled`), `assignee`, `due_date`, `version` (оптимистичная блокировка), `created_at`, `updated_at`, `deleted_at` (soft delete).
- **История изменений**: `id`, `task_id`, `field_name`, `old_value`, `new_value`, `changed_at`.
- **Outbox** (будет добавлена позже): `id`, `aggregate_id`, `event_type`, `payload`, `created_at`, `processed_at`.
- **Аналитика** (ClickHouse, позже): денормализованные данные для быстрых агрегатов.

**Фронтенд:** React + Vite + Ant Design (UI Kit), подключение через WebSocket к real-time событиям.

---

## 🗺 Development Roadmap (как создавался проект)

Проект разрабатывается **эволюционно**: каждый этап добавляет новую технологию или функциональность, не ломая предыдущие.  
Каждый шаг соответствует **отдельному коммиту** в репозитории (см. историю коммитов).

---

### 📌 Шаг 0 – Базовое CRUD + Sentry + React/AntD

**Что сделано:**

1. REST API на Go (Gin) с полным CRUD для задач.
2. PostgreSQL с миграциями (`golang-migrate`).
3. Транзакции: при обновлении задачи изменяется `version` (оптимистичная блокировка) и пишется история в `task_history` в одной транзакции.
4. Мягкое удаление (`deleted_at`).
5. Фронтенд на React + Vite + Ant Design (таблица, фильтры, модальные формы).
6. Sentry подключён к бэкенду и фронтенду.

**Коммит:** `feat(step0): init project with CRUD, migrations, transactions, Sentry, React+AntD`

---

## 🚀 Запуск проекта (шаг 0)

### 1️⃣ Переменные окружения

Создай файл `.env` в корне проекта (или экспортируй переменные в терминале):

```env
DB_URL=postgres://postgres:postgres@localhost:5433/taskflow?sslmode=disable
SERVER_PORT=8080
SENTRY_DSN=https://your-dsn@sentry.io/your-project