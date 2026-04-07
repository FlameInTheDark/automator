.DEFAULT_GOAL := build

APP_NAME := automator$(if $(filter Windows_NT,$(OS)),.exe,)
WEB_DIR := web
WEB_DEPS := $(WEB_DIR)/node_modules/.package-lock.json
EMBED_DIR := internal/api/web/dist

export CGO_ENABLED := 1

ifeq ($(OS),Windows_NT)
RM_BINARY = if exist "$(APP_NAME)" del /q "$(APP_NAME)"
RM_EMBED = if exist "$(EMBED_DIR)" rmdir /s /q "$(EMBED_DIR)"
else
RM_BINARY = rm -f "$(APP_NAME)"
RM_EMBED = rm -rf "$(EMBED_DIR)"
endif

.PHONY: build build-web run clean test lint docker docker-run

$(WEB_DEPS): $(WEB_DIR)/package.json $(WEB_DIR)/package-lock.json
	cd $(WEB_DIR) && npm ci

build-web: $(WEB_DEPS)
	cd $(WEB_DIR) && npm run build

build: build-web
	go build -ldflags="-s -w" -o $(APP_NAME) ./cmd/server

run: build-web
	go run ./cmd/server

clean:
	$(RM_BINARY)
	$(RM_EMBED)

test:
	go test -race -cover ./...

lint:
	golangci-lint run

docker:
	docker build -t automator .

docker-run:
	docker-compose up -d
