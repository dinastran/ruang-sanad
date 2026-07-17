APP_NAME   := laju-go
BINARY     := $(APP_NAME)
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS    := -ldflags="-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

# Windows: pake WSL (wsl make build) atau winget install GnuWin32.Make
ifeq ($(OS),Windows_NT)
BINARY := $(APP_NAME).exe
endif

.PHONY: build build-go build-linux verify test lint generate templ clean docker version migrate db-generate db-refresh

build: vite-build go-build

build-go: go-build

build-linux:
	GOOS=linux GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(BINARY) ./cmd/laju-go

go-build:
	go build -trimpath $(LDFLAGS) -o $(BINARY) ./cmd/laju-go

vite-build:
	npm run build

verify:
	npm run verify

go test ./...

test:
	go test ./...

lint:
	golangci-lint run ./...

generate: templ db-generate

templ:
	templ generate

db-generate:
	sqlc generate

migrate:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations sqlite3 ./data/app.db up

db-refresh:
	rm -f ./data/app.db ./data/app.db-shm ./data/app.db-wal ./data/app.db-wal2
	rm -rf ./data/cache

clean:
	rm -rf $(BINARY) tmp/ dist/ data/*.db* data/cache

docker:
	docker build -t $(APP_NAME) .

version:
	@echo "$(VERSION) (commit: $(COMMIT))"
