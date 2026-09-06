.PHONY: setup doctor server up down down-to redo status reset build dev vet test checkout opencode tmux generate

vet:
	go build ./...
	go vet ./...
	cd web && npx tsc -b

test:
	go test ./...

setup:
	go run ./cmd/pr-review setup

doctor:
	go run ./cmd/pr-review doctor

server:
	go run ./cmd/pr-review serve

up:
	go run ./cmd/pr-review migrate up

down:
	go run ./cmd/pr-review migrate down

down-to:
	go run ./cmd/pr-review migrate down-to $(V)

redo:
	go run ./cmd/pr-review migrate redo

status:
	go run ./cmd/pr-review migrate status

reset:
	go run ./cmd/pr-review migrate reset

generate:
	go tool sqlc generate

checkout:
	go run ./cmd/checkout $(ARGS)

opencode:
	go run ./cmd/opencode $(ARGS)

tmux:
	go run ./cmd/tmux $(ARGS)

dev:
	cd web && npm run dev

build:
	cd web && npm run build
	rm -rf cmd/pr-review/dist && cp -r web/dist cmd/pr-review/dist
	go build -o bin/pr-review ./cmd/pr-review
