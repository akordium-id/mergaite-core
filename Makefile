.PHONY: run test build sqlc docker-up docker-down migrate-up migrate-down

export PATH := $(PATH):$(HOME)/go/bin

DB_URL ?= postgres://mergaite:mergaite_password@localhost:5434/mergaite_core?sslmode=disable

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test -v ./...

sqlc:
	sqlc generate

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1
