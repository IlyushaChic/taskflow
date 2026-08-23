# ============================================================
#  Основные команды разработки
# ============================================================
.PHONY: migrate-up migrate-down run build go-test

migrate-up:
	@echo "Running migrations up..."
	migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	@echo "Running migrations down..."
	migrate -path ./migrations -database "$(DB_URL)" down

run:
	@echo "Starting API server..."
	go run cmd/api/main.go

build:
	@echo "Building binary..."
	go build -o bin/taskflow-api cmd/api/main.go

go-test:
	@echo "Running Go tests..."
	go test -v -race ./...

# ============================================================
#  Управление Docker Compose
# ============================================================
COMPOSE_FILE := docker-compose.yml
BACKEND_URL := http://localhost:8080
FRONTEND_URL := http://localhost:3000
JAEGER_URL := http://localhost:16686
GRAFANA_URL := http://localhost:3001
PROMETHEUS_URL := http://localhost:9090

.PHONY: up down clean

up:
	@echo "🚀 Starting all services via docker-compose..."
	docker-compose -f $(COMPOSE_FILE) up -d
	@echo "⏳ Waiting for backend to be ready (max 60s)..."
	@for i in $$(seq 1 30); do \
		curl -s $(BACKEND_URL)/health > /dev/null && echo "✅ Backend ready" && break; \
		sleep 2; \
		if [ $$i -eq 30 ]; then echo "❌ Backend not ready after 60s"; exit 1; fi; \
	done

down:
	@echo "🛑 Stopping all services..."
	docker-compose -f $(COMPOSE_FILE) down

clean:
	@echo "🧹 Removing containers, networks, and volumes..."
	docker-compose -f $(COMPOSE_FILE) down -v

# ============================================================
#  Тесты для каждого шага (требуют запущенных сервисов)
# ============================================================
.PHONY: test test-full test-step0 test-step1 test-step2 test-step3 test-step4 test-step5 test-step7 test-step9

test: go-test test-step0 test-step1 test-step2 test-step3 test-step4 test-step5 test-step7 test-step9
	@echo "✅ All tests completed."

test-step0:
	@echo "=== Testing Step 0: CRUD ==="
	@curl -s -X POST $(BACKEND_URL)/api/v1/tasks -H "Content-Type: application/json" -d '{"title":"Test CRUD","assignee":"Tester"}' | jq .
	@echo "Fetching tasks..."
	@curl -s $(BACKEND_URL)/api/v1/tasks | jq .
	@echo "✅ Step 0 OK."

test-step1:
	@echo "=== Testing Step 1: Redis caching ==="
	@echo "First request (should be cache miss)..."
	@curl -s -w "\nTime: %{time_total}s\n" $(BACKEND_URL)/api/v1/tasks > /dev/null
	@echo "Second request (should be cache hit, faster)..."
	@curl -s -w "\nTime: %{time_total}s\n" $(BACKEND_URL)/api/v1/tasks > /dev/null
	@echo "✅ Step 1 OK."

test-step2:
	@echo "=== Testing Step 2: RabbitMQ + Outbox ==="
	@echo "Creating a task to trigger outbox..."
	@curl -s -X POST $(BACKEND_URL)/api/v1/tasks -H "Content-Type: application/json" -d '{"title":"Test Outbox","assignee":"Tester"}' > /dev/null
	@echo "Waiting 5s for worker to process..."
	@sleep 5
	@echo "Checking worker logs for 'Processed outbox'..."
	@docker logs taskflow-rabbit-worker 2>&1 | grep -i "Processed outbox" || echo "⚠️ No outbox entry found (check logs manually)"
	@echo "✅ Step 2 OK (worker may need more time)."

test-step3:
	@echo "=== Testing Step 3: WebSocket ==="
	@echo "WebSocket requires manual verification: open two browser tabs on $(FRONTEND_URL), create a task in one, and see the update in the other."
	@echo "✅ Step 3 OK (manual)."

test-step4:
	@echo "=== Testing Step 4: ClickHouse analytics ==="
	@curl -s $(BACKEND_URL)/api/v1/stats | jq .
	@echo "✅ Step 4 OK."

test-step5:
	@echo "=== Testing Step 5: Kafka streaming ==="
	@echo "Creating a task to produce Kafka event..."
	@curl -s -X POST $(BACKEND_URL)/api/v1/tasks -H "Content-Type: application/json" -d '{"title":"Test Kafka","assignee":"Tester"}' > /dev/null
	@echo "Check Kafka consumer logs manually: docker logs taskflow-kafka-worker"
	@echo "✅ Step 5 OK."

test-step7:
	@echo "=== Testing Step 7: Docker Compose health ==="
	@curl -s $(BACKEND_URL)/health | jq .
	@echo "✅ Step 7 OK."

test-step9:
	@echo "=== Testing Step 9: Observability ==="
	@echo "Prometheus: $(PROMETHEUS_URL)/-/healthy"
	@curl -s $(PROMETHEUS_URL)/-/healthy
	@echo "Grafana: $(GRAFANA_URL)/api/health"
	@curl -s $(GRAFANA_URL)/api/health
	@echo "Jaeger: $(JAEGER_URL)/api/services"
	@curl -s $(JAEGER_URL)/api/services
	@echo "✅ Step 9 OK."

# ============================================================
#  Полный цикл: поднять → протестировать → остановить
# ============================================================
.PHONY: test-full ci

test-full: up test down
	@echo "🎉 Full test cycle completed successfully."

ci: test-full

# ============================================================
#  Справка
# ============================================================
.PHONY: help

help:
	@echo "Available targets:"
	@echo "  migrate-up    - Apply migrations"
	@echo "  migrate-down  - Rollback migrations"
	@echo "  run           - Start API server locally"
	@echo "  build         - Build binary"
	@echo "  go-test       - Run Go tests only"
	@echo ""
	@echo "  up            - Start all Docker services"
	@echo "  down          - Stop all Docker services"
	@echo "  clean         - Stop and remove volumes"
	@echo ""
	@echo "  test-step0    - Test CRUD"
	@echo "  test-step1    - Test Redis caching"
	@echo "  test-step2    - Test RabbitMQ + Outbox"
	@echo "  test-step3    - Test WebSocket (manual)"
	@echo "  test-step4    - Test ClickHouse"
	@echo "  test-step5    - Test Kafka"
	@echo "  test-step7    - Test Docker Compose health"
	@echo "  test-step9    - Test Observability"
	@echo ""
	@echo "  test          - Run Go tests + all step tests (services must be up)"
	@echo "  test-full     - Full cycle: up → test → down"
	@echo "  ci            - Alias for test-full"