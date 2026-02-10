.PHONY: up down build test lint migrate seed loadtest

up:
	docker compose -f deploy/docker-compose.yml up -d --build

down:
	docker compose -f deploy/docker-compose.yml down -v

build:
	docker compose -f deploy/docker-compose.yml build

test:
	cd backend && go test ./...
	cd frontend && npm test -- --runInBand

lint:
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint

migrate:
	cd backend && go run ./cmd/server migrate

seed:
	cd backend && go run ./cmd/server seed

loadtest:
	cd scripts && ./loadtest.sh
