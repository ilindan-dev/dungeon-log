BIN_DIR=bin
APP_NAME=dungeon-log

.PHONY: build
build:
	go build -o $(BIN_DIR)/$(APP_NAME) cmd/$(APP_NAME)/main.go

.PHONY: run
run:
	go run cmd/$(APP_NAME)/main.go

.PHONY: test
test:
	go test -v -race ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: tools
tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.12.0