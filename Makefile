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

dev:
	cd web && npm run dev

build:
	cd web && npm run build
	rm -rf cmd/server/dist && cp -r web/dist cmd/server/dist
	go build -o bin/server ./cmd/server
	go build -o bin/migrate ./cmd/migrate
