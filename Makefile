include .env

docker-shell:
	docker exec -it dev-$(APP_NAME)-app-1 sh

dev:
	make dev-build
	make dev-run

dev-run:
	docker compose -f docker-compose.dev.yaml -p dev-$(APP_NAME) up

dev-build:
	docker compose -f docker-compose.dev.yaml -p dev-$(APP_NAME) build

dev-down:
	docker compose -f docker-compose.dev.yaml -p dev-$(APP_NAME) down -v

prod-build:
	docker compose -f docker-compose.yaml -p prod-$(APP_NAME) build

prod-export:
	docker save -o tmp/$(APP_NAME).tar prod-$(APP_NAME)-app

prod-run:
	docker compose -f docker-compose.yaml -p prod-$(APP_NAME) up

prod-down:
	docker compose -f docker-compose.yaml -p prod-$(APP_NAME) down -v

migrate-new:
	GOOSE_MIGRATION_DIR=./migrations goose create $(name) sql

migrate-up:
	GOOSE_MIGRATION_DIR=./migrations goose postgres $(DB_URI) up

migrate-down:
	GOOSE_MIGRATION_DIR=./migrations goose postgres $(DB_URI) down

migrate-fresh:
	GOOSE_MIGRATION_DIR=./migrations goose postgres $(DB_URI) down-to 0
	GOOSE_MIGRATION_DIR=./migrations goose postgres $(DB_URI) up
