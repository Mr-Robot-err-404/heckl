.PHONY: run sandbox-setup sandbox-doctor sandbox-serve sandbox-clean setup doctor server up down down-to redo status reset build dev vet test checkout opencode tmux generate build-server

GO_BUILD = go build -o bin/heckl ./cmd/heckl

vet:
	go build ./...
	go vet ./...
	cd web && npx tsc -b

test:
	go test ./...

setup:
	go run ./cmd/heckl setup

doctor:
	go run ./cmd/heckl doctor

server:
	go run ./cmd/heckl serve

up:
	go run ./cmd/heckl migrate up

down:
	go run ./cmd/heckl migrate down

down-to:
	go run ./cmd/heckl migrate down-to $(V)

redo:
	go run ./cmd/heckl migrate redo

status:
	go run ./cmd/heckl migrate status

reset:
	go run ./cmd/heckl migrate reset

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

build-server:
	$(GO_BUILD)

build:
	cd web && npm run build
	rm -rf cmd/heckl/dist && cp -r web/dist cmd/heckl/dist
	$(GO_BUILD)

run:
	./bin/heckl serve

SANDBOX ?= /tmp/heckl-sandbox
SANDBOX_ENV = XDG_CONFIG_HOME=$(SANDBOX)/.config \
	XDG_DATA_HOME=$(SANDBOX)/.local/share \
	GH_CONFIG_DIR=$(SANDBOX)/gh \
	GITHUB_TOKEN=

sandbox-setup:
	mkdir -p $(SANDBOX)
	env $(SANDBOX_ENV) ./bin/heckl setup

sandbox-doctor:
	env $(SANDBOX_ENV) ./bin/heckl doctor

sandbox-serve:
	env $(SANDBOX_ENV) ./bin/heckl serve

sandbox-clean:
	rm -rf $(SANDBOX)
