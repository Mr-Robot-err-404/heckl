.PHONY: run sandbox-setup sandbox-doctor sandbox-serve sandbox-clean setup doctor server up down down-to redo status reset build dev vet test checkout opencode tmux generate

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

run:
	./bin/pr-review serve

SANDBOX ?= /tmp/pr-review-sandbox
SANDBOX_ENV = XDG_CONFIG_HOME=$(SANDBOX)/.config \
	XDG_DATA_HOME=$(SANDBOX)/.local/share \
	GH_CONFIG_DIR=$(SANDBOX)/gh \
	GITHUB_TOKEN=

sandbox-setup:
	mkdir -p $(SANDBOX)
	env $(SANDBOX_ENV) ./bin/pr-review setup

sandbox-doctor:
	env $(SANDBOX_ENV) ./bin/pr-review doctor

sandbox-serve:
	env $(SANDBOX_ENV) ./bin/pr-review serve

sandbox-clean:
	rm -rf $(SANDBOX)
