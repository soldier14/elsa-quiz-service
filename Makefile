SHELL := /bin/bash

GO ?= go
GOTOOLCHAIN ?= local
CGO_ENABLED ?= 0

.PHONY: build test lint fmt fmtcheck docker docker-run

build:
	@echo "Building binary..."
	CGO_ENABLED=$(CGO_ENABLED) GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) build -o bin/quiz-service ./cmd

test:
	@echo "Running tests..."
	CGO_ENABLED=$(CGO_ENABLED) GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) test ./...

lint:
	@echo "Running go vet..."
	CGO_ENABLED=$(CGO_ENABLED) GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) vet ./...

fmt:
	@echo "Formatting..."
	gofmt -w cmd internal

fmtcheck:
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l cmd internal)" || (echo "gofmt needed on:" && gofmt -l cmd internal && exit 1)

docker:
	@echo "Building Docker image..."
	docker build -t elsa-quiz-service:latest .

docker-run:
	@echo "Running Docker image..."
	docker run --rm -p 8080:8080 -e CONFIG_PATH=/app/config/config.yaml elsa-quiz-service:latest start
