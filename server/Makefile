.PHONY: migrate-up migrate-down migrate-status generate run

include .env
export

migrate-up:
	goose postgres $(DATABASE_URL) up -dir internal/database/migrations

migrate-down:
	goose postgres $(DATABASE_URL) down -dir internal/database/migrations

migrate-status:
	goose postgres $(DATABASE_URL) status -dir internal/database/migrations

generate:
	sqlc generate

run:
	air
