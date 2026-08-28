.PHONY: server up down status reset build

server:
	go run ./cmd/server

up:
	go run ./cmd/migrate up

down:
	go run ./cmd/migrate down

status:
	go run ./cmd/migrate status

reset:
	go run ./cmd/migrate reset

build:
	go build -o bin/server ./cmd/server
	go build -o bin/migrate ./cmd/migrate
