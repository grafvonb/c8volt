BINARY := c8volt
BIN_DIR := bin
PKG := ./...
COVER_DIR := .coverage
COVER_OUT := $(COVER_DIR)/coverage.out
COVER_HTML := $(COVER_DIR)/coverage.html

.PHONY: all tidy generate build test lint fmt vet clean install run cover cover.html release docs

all: tidy fmt vet lint test build docs

tidy:
	go mod tidy

generate:
	go generate $(PKG)

docs:
	go run ./docsgen -out ./docs/cli -format markdown

build:
	mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/$(BINARY) .

install:
	go install .

run: build
	./$(BIN_DIR)/$(BINARY) --help

test:
	go test $(PKG) -race -count=1

lint:
	golangci-lint run

fmt:
	go fmt $(PKG)

vet:
	go vet $(PKG)

clean:
	rm -rf $(BIN_DIR) $(COVER_DIR)

# Coverage
cover:
	mkdir -p $(COVER_DIR)
	go test $(PKG) -race -covermode=atomic -coverprofile=$(COVER_OUT)
	go tool cover -func=$(COVER_OUT) | tail -n 1

cover.html: cover
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)
	@echo "Open $(COVER_HTML)"

# Delegate to GoReleaser
release:
	goreleaser release --clean --skip=publish
