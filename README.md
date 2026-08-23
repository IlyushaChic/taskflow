# 📌 О проекте

**TaskFlow** — это менеджер задач, который растёт от простого CRUD-приложения до распределённой системы с асинхронностью, аналитикой, стримингом, оркестрацией и полным observability-стеком.

---

# 🗺 Development Roadmap (как создавался проект)

Проект разрабатывается **эволюционно**: каждый этап добавляет новую технологию или функциональность, не ломая предыдущие.  
Каждый шаг соответствует **отдельному коммиту** в репозитории.

---

### 📌 Шаг 0 – Базовое CRUD + React/AntD

**Что сделано:**
1. REST API на Go (Gin) с полным CRUD для задач.
2. PostgreSQL с миграциями (`golang-migrate`).
3. Транзакции: при обновлении задачи изменяется `version` (оптимистичная блокировка) и пишется история в `task_history` в одной транзакции.
4. Мягкое удаление (`deleted_at`).
5. Фронтенд на React + Vite + Ant Design (таблица, фильтры, модальные формы).

**Коммит:** `feat(step0): init project with CRUD, migrations, transactions, React+AntD`

---

### 📌 Шаг 1 – Redis (кеширование)

**Что сделано:**
- Кеширование списка задач (`GET /tasks`) на 1 минуту.
- Кеширование одной задачи (`GET /tasks/:id`) на 30 секунд.
- Инвалидация кеша при создании, обновлении или удалении задачи.
- Логирование `cache hit` / `cache miss`.

**Как проверить:** дважды выполнить `GET /tasks` – первый `cache miss`, второй `cache hit`.

**Коммит:** `feat(step1): add Redis caching`

---

### 📌 Шаг 2 – RabbitMQ + Transactional Outbox

**Что сделано:**
- Таблица `outbox` для надёжной публикации событий.
- В транзакции обновления задачи сохраняется событие в outbox.
- Воркер `cmd/workers/rabbit-consumer` каждые 5 секунд забирает необработанные сообщения, отправляет их в RabbitMQ (очередь `email_notifications`) и помечает как обработанные.
- Имитация отправки email (лог).

**Как проверить:** запустить RabbitMQ и воркер, создать/обновить задачу – в логах воркера появится сообщение.

**Коммит:** `feat(step2): add RabbitMQ with transactional outbox pattern`

---

### 📌 Шаг 3 – WebSockets (real‑time)

**Что сделано:**
- Эндпоинт `/ws` для WebSocket-соединений.
- `Hub` для хранения активных соединений и рассылки событий.
- При изменении задачи всем клиентам отправляется событие (`task_created`, `task_updated`, `task_deleted`).
- Фронтенд подключается к WebSocket, показывает уведомления и обновляет таблицу.

**Как проверить:** открыть две вкладки браузера, в одной создать/обновить задачу – во второй появится уведомление и таблица обновится.

**Коммит:** `feat(step3): add WebSocket real-time updates`

---

### 📌 Шаг 4 – ClickHouse (аналитика)

**Что сделано:**
- Таблица `task_analytics` в ClickHouse (MergeTree) с полями: `task_id`, `event_type`, `status`, `assignee`, `timestamp`.
- Асинхронная запись событий в ClickHouse с использованием Batch Insert (пакеты по 100 или каждые 5 секунд).
- Эндпоинт `GET /api/v1/stats` – количество задач по статусам за последние 24 часа.

**Как проверить:** запустить ClickHouse, создать таблицу, выполнить несколько изменений задач, подождать 5 секунд, запросить `/stats`.

**Коммит:** `feat(step4): add ClickHouse for analytics`

---

### 📌 Шаг 5 – Apache Kafka (стриминг)

**Что сделано:**
- Продюсер Kafka публикует события изменения задач в топик `task-events`.
- Консьюмер (воркер) читает из Kafka и пишет в ClickHouse, заменяя прямую запись из шага 4.
- Партиционирование по `task_id`.

**Как проверить:**
- Запустить Kafka (Docker), создать топик `task-events`.
- Запустить консьюмера, затем бэкенд.
- Создать/обновить задачу – проверить логи продюсера и консьюмера, а также данные в ClickHouse.

**Коммит:** `feat(step5): add Kafka producer/consumer for event streaming`

---

### 📌 Шаг 6 – Temporal (долгие процессы) (в разработке)

**Статус:** Инфраструктура Temporal требует отдельной конфигурации. В текущей версии проекта реализация отложена. Планируется добавить в будущем.

---

### 📌 Шаг 7 – Docker-контейнеризация

**Что сделано:**
- Multi-stage Dockerfile для бэкенда и воркеров.
- Dockerfile для фронтенда (сборка через Vite, раздача через nginx).
- `docker-compose.yml` с PostgreSQL, Redis, RabbitMQ, ClickHouse, Kafka+Zookeeper, бэкендом, воркерами и фронтендом.
- Автоматический накат миграций при старте через отдельный сервис `migrate`.
- Все сервисы запускаются одной командой.

**Как проверить:**
- Выполнить `docker-compose up -d`.
- Открыть `http://localhost:3000` – фронт.
- Проверить API: `curl http://localhost:8080/health`.

**Коммит:** `feat(step7): dockerize all services with compose`

---

### 📌 Шаг 8 – Kubernetes + Helm (оркестрация)

**Что сделано:**
- Создан Helm-чарт `taskflow` со структурой деплойментов, сервисов, ConfigMap, Secret и PVC.
- Настроены `livenessProbe` и `readinessProbe` для бэкенда.
- Использован `values.yaml` для параметризации: реплики, образы, порты, переменные окружения.
- Чарт развёртывает все основные сервисы: бэкенд, фронтенд, PostgreSQL, Redis, ClickHouse, RabbitMQ.
- **Примечание:** Kafka, Zookeeper и воркеры временно исключены из-за ограничений ресурсов в локальном кластере, но могут быть добавлены при необходимости.
- Проверено на minikube.

**Как проверить:**
- Установить чарт: `helm install taskflow ./helm/taskflow -n taskflow --create-namespace`.
- Проверить поды: `kubectl get pods -n taskflow`.
- Получить доступ к API через `kubectl port-forward -n taskflow svc/taskflow-backend 8080:8080`.
- Проверить работу фронта через `kubectl port-forward -n taskflow svc/taskflow-frontend 3000:80`.

**Коммит:** `feat(step8): add Kubernetes manifests and Helm chart`

---

### 📌 Шаг 9 – Prometheus + Grafana + Jaeger (observability) (локально)

**Что сделано:**
- Внедрён OpenTelemetry в бэкенд (локально):
  - TracerProvider с экспортером в Jaeger (OTLP).
  - Middleware `otelgin` для автоматической трассировки HTTP-запросов.
  - Кастомные спаны для запросов к БД, Redis, RabbitMQ/Kafka.
- Добавлен эндпоинт `/metrics` для сбора метрик Prometheus.
- В `docker-compose.yml` добавлены сервисы:
  - **Jaeger** – сбор и визуализация трейсов (порт 16686).
  - **Prometheus** – сбор метрик (порт 9090).
  - **Grafana** – визуализация метрик и трейсов (порт 3001, логин admin/admin).
- Настроены источники данных для Grafana (Prometheus и Jaeger) через provisioning.
- В Helm-чарт также добавлены деплойменты и сервисы для Jaeger, Prometheus, Grafana.

> **Примечание:** в CI/CD observability временно отключена из-за устаревших пакетов, но локально работает при правильной настройке.

**Как проверить:**
- Запустить стек через `docker-compose up -d`.
- Отправить несколько запросов к API (например, создать задачу).
- Зайти в Jaeger UI: http://localhost:16686 – выбрать сервис `taskflow-api` и увидеть трейсы.
- Зайти в Grafana: http://localhost:3001 (admin/admin) – в разделе Explore построить графики по метрикам.
- Проверить эндпоинт `/metrics`: `curl http://localhost:8080/metrics`.

**Коммит:** `feat(step9): add Prometheus, Grafana, Jaeger with OpenTelemetry`

---

### 📌 Шаг 10 – CI/CD (GitHub Actions)

**Что сделано:**
- Написан workflow `.github/workflows/ci-cd.yaml`.
- Jobs:
  - **lint** – проверка кода через `golangci-lint` (установлен через `go install` для совместимости с Go 1.26.3).
  - **test** – запуск тестов с покрытием.
  - **build-and-push** – сборка Docker-образов для backend, frontend, rabbit-worker, kafka-worker и публикация в GitHub Container Registry (GHCR).
  - **deploy** (закомментирован, т.к. нет удалённого кластера) – автоматический деплой в Kubernetes через Helm.
- Автоматический запуск при пуше в `main` или создании тега `v*`.
- Используются секреты: `KUBE_CONFIG` (для деплоя, пока не используется), `DB_PASSWORD`, `RABBITMQ_PASSWORD`.

**Как проверить:**
- Запушить коммит в `main` – перейти на вкладку Actions и убедиться, что пайплайн выполнился успешно.
- Проверить, что новый образ появился в GitHub Container Registry.

**Коммит:** `feat(step10): add CI/CD pipeline with GitHub Actions`

---

## 🚀 Запуск проекта (локально, в Docker и в Kubernetes)

### 🐳 Запуск через Docker Compose (с Observability)

1. В корне проекта выполни:

```bash
docker-compose up -d
```

2. Открой фронт: http://localhost:3000.
3. Проверь API: curl http://localhost:8080/health.
4. Проверь метрики: curl http://localhost:8080/metrics.
5. Открой Jaeger UI: http://localhost:16686.
6. Открой Grafana: http://localhost:3001 (логин: admin, пароль: admin).

☸️ Запуск в Kubernetes (с помощью Helm)
Загрузи образы в кластер (если локальные образы):

```bash
minikube image load taskflow-backend:latest
minikube image load taskflow-frontend:latest
```

Установи чарт:

```bash
helm install taskflow ./helm/taskflow -n taskflow --create-namespace
```
Проверь, что поды запустились:

```bash
kubectl get pods -n taskflow
```
Получи доступ к API:

```bash
kubectl port-forward -n taskflow svc/taskflow-backend 8080:8080
```
Открой фронтенд:

```bash
kubectl port-forward -n taskflow svc/taskflow-frontend 3000:80
```



🗄️ Миграции
Миграции разделены по базам данных:

PostgreSQL – основная БД (таблицы tasks, task_history, outbox).
Путь: migrations/postgres/

ClickHouse – аналитика (таблица task_analytics).
Путь: migrations/clickhouse/

Применение миграций PostgreSQL
Локально (через golang-migrate):

```bash
export DB_URL="postgres://postgres:postgres@localhost:5433/taskflow?sslmode=disable"
migrate -path ./migrations/postgres -database "$DB_URL" up
```
В Docker Compose:
Миграции PostgreSQL применяются автоматически при запуске через сервис migrate

Применение миграций ClickHouse
```bash
docker exec -i taskflow-clickhouse clickhouse-client --user default --password '' < migrations/clickhouse/20260101000000_create_task_analytics.up.sql
```

🔧 Локальный запуск (разработка)
1️⃣ Переменные окружения
Создай файл .env в корне проекта:

env
DB_URL=postgres://postgres:postgres@localhost:5433/taskflow?sslmode=disable
SERVER_PORT=8080
REDIS_URL=redis://localhost:6379/0
CLICKHOUSE_DSN=localhost:9000
KAFKA_BROKERS=localhost:9092
JAEGER_ENDPOINT=http://localhost:14268/api/traces

2️⃣ Запуск баз через Docker (отдельно)
```bash
docker run -d --name postgres-local -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=taskflow -p 5433:5432 postgres:15-alpine
docker run -d --name redis-local -p 6379:6379 redis:7-alpine
docker run -d --name rabbitmq-local -p 5672:5672 -p 15672:15672 rabbitmq:3-management-alpine
docker run -d --name clickhouse-local -p 8123:8123 -p 9000:9000 -e CLICKHOUSE_DEFAULT_PASSWORD="" clickhouse/clickhouse-server:latest
docker run -d --name jaeger-local -p 16686:16686 -p 14268:14268 jaegertracing/all-in-one:latest
```
3️⃣ Применение миграций PostgreSQL
```bash
export DB_URL="postgres://postgres:postgres@localhost:5433/taskflow?sslmode=disable"
migrate -path ./migrations/postgres -database "$DB_URL" up
```
4️⃣ Запуск бэкенда
```bash
export DB_URL="postgres://postgres:postgres@localhost:5433/taskflow?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export CLICKHOUSE_DSN="localhost:9000"
export KAFKA_BROKERS="localhost:9092"
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
go run cmd/api/main.go
```
5️⃣ Запуск фронтенда
```bash
cd frontend
npm install
npm run dev
```
Фронт будет доступен на http://localhost:5173.

🧪 Проверка работы API
```bash
# Health
curl http://localhost:8080/health

# Список задач
curl http://localhost:8080/api/v1/tasks

# Создание задачи
curl -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d '{"title":"Test","assignee":"Me"}'

# Статистика (ClickHouse)
curl http://localhost:8080/api/v1/stats

# Метрики Prometheus
curl http://localhost:8080/metrics
```

Observability

Jaeger UI: http://localhost:16686 – трейсы распределённых запросов (локально).
Prometheus: http://localhost:9090 – метрики.
Grafana: http://localhost:3001 (admin/admin) – дашборды.

WebSocket
Открой две вкладки браузера на http://localhost:3000 (или 5173). Создай задачу в одной – во второй появится уведомление и таблица обновится.


🗄️ Миграции в Kubernetes
Если миграции не накатились автоматически, примени их вручную через под PostgreSQL:

```bash
cat migrations/postgres/20260101000000_create_tasks_table.up.sql | kubectl exec -i -n taskflow <pod-postgres> -- psql -U postgres -d taskflow
cat migrations/postgres/20260101000001_create_task_history_table.up.sql | kubectl exec -i -n taskflow <pod-postgres> -- psql -U postgres -d taskflow
cat migrations/postgres/20260101000002_create_outbox_table.up.sql | kubectl exec -i -n taskflow <pod-postgres> -- psql -U postgres -d taskflow

```
Замени <pod-postgres> на имя пода PostgreSQL (можно узнать через kubectl get pods -n taskflow).

🛠️ Автоматизация с Makefile
В проекте есть Makefile, который облегчает управление и тестирование:

```bash
make help          # Показать все доступные цели
make up            # Запустить все сервисы через Docker Compose
make down          # Остановить все сервисы
make clean         # Остановить и удалить все volumes
make test          # Запустить Go-тесты и все проверки шагов (сервисы должны быть запущены)
make test-full     # Полный цикл: up → test → down
make test-step0    # Проверить только CRUD
make test-step1    # Проверить Redis кеширование
```
# ... и так далее для шагов 2-9 (кроме 6 и 8)

Каждый шаг зафиксирован отдельным коммитом. 🚀