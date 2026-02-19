APP_NAME := document-api

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build -o ./bin/$(APP_NAME) ./cmd/api

.PHONY: run
run:
	go run ./cmd/api --serve

.PHONY: generate
generate:
	go generate ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: compose-up
compose-up:
	docker compose up -d

.PHONY: compose-down
compose-down:
	docker compose down -v

.PHONY: docker-build
docker-build:
	docker build -t $(APP_NAME):local .

.PHONY: docker-run
docker-run:
	docker run --rm -p 8080:8080 --env-file ./.env $(APP_NAME):local
