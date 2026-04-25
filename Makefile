SHELL := /bin/sh

GO_IMAGE := golang:1.22-alpine
REQUIRED_COVERAGE := 100.0%
BACKEND_IMAGE := pack-shipping-calculator-backend:local
FRONTEND_IMAGE := pack-shipping-calculator-frontend:local

.DEFAULT_GOAL := help

.PHONY: help
help:
	@printf '%s\n' 'Pack Shipping Calculator:'
	@printf '%s\n' ''
	@printf '%s\n' '  make dev       Clean Docker state, build, and run the app'
	@printf '%s\n' '  make validate  Clean Docker state, run all checks, and build images'
	@printf '%s\n' '  make down      Stop the app'
	@printf '%s\n' '  make reset     Stop the app and remove persisted Docker data'
	@printf '%s\n' '  make clean     Remove generated local artifacts and local Docker images'

.PHONY: dev
dev: clean-docker
	docker compose up --build

.PHONY: validate
validate: clean-docker
	docker run --rm -v $(CURDIR):/workspace -w /workspace/backend $(GO_IMAGE) go test ./... -covermode=count -coverprofile=coverage.out
	docker run --rm -v $(CURDIR):/workspace -w /workspace/backend $(GO_IMAGE) go tool cover -func=coverage.out
	@coverage="$$(docker run --rm -v $(CURDIR):/workspace -w /workspace/backend $(GO_IMAGE) sh -c "go tool cover -func=coverage.out | awk '/^total:/ { print \$$3 }'")"; \
	if [ "$$coverage" != "$(REQUIRED_COVERAGE)" ]; then \
		printf 'coverage must be %s, got %s\n' "$(REQUIRED_COVERAGE)" "$$coverage"; \
		exit 1; \
	fi
	docker run --rm -v $(CURDIR):/workspace -w /workspace/backend $(GO_IMAGE) go vet ./...
	docker build --pull -f Dockerfile -t $(BACKEND_IMAGE) .
	docker build --pull -f frontend/Dockerfile -t $(FRONTEND_IMAGE) .
	docker compose config

.PHONY: down
down:
	docker compose down

.PHONY: reset
reset:
	docker compose down -v

.PHONY: clean
clean: clean-docker
	rm -f backend/coverage.out
	rm -rf backend/bin frontend/dist

.PHONY: clean-docker
clean-docker:
	docker compose down -v --remove-orphans --rmi local
	docker image rm -f $(BACKEND_IMAGE) $(FRONTEND_IMAGE) 2>/dev/null || true
