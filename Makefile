SHELL := /bin/bash
.DEFAULT_GOAL := help

.PHONY: help venv-uv venv-pip install-uv install-dev-uv install-pip install-dev-pip \
        test ruff black lint clean

help:
	@echo ""
	@echo "Targets:"
	@echo "  make build-docker      Build a Docker image"
	@echo "  make up        		Start the Docker container using latest image"
	@echo "  make down    			Shut down the running container"
	@echo "  make logs    			Show the logs of the running container"
	@echo ""
	@echo "  make test              Run tests"
	@echo ""

build-docker:
	./build.sh docker

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs --follow

test:
	go clean -cache
	go test ./... -v
