.PHONY: migrate-up migrate-down run build test

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

test:
	@echo "Running tests..."
	go test -v -race ./...

