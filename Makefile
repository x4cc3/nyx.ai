GOCACHE ?= $(CURDIR)/.gocache
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X nyx/internal/version.Version=$(VERSION) -X nyx/internal/version.Commit=$(COMMIT) -X nyx/internal/version.BuildDate=$(BUILD_DATE)
GO := GOCACHE=$(GOCACHE) go

.PHONY: fmt test integration-test executor-smoke build build-api build-orchestrator build-executor build-migrate run-api run-orchestrator run-executor migrate rollback infra-up infra-down docker-build-api docker-build-orchestrator docker-build-executor docker-build-executor-pentest docker-build-web docker-build-stack web-install web-dev web-build compose-prod-up compose-prod-down lint vet coverage clean security-scan

fmt:
	gofmt -w cmd internal

test:
	$(GO) test ./...

integration-test:
	$(GO) test -tags=integration ./internal/integration/...

executor-smoke:
	NYX_RUN_DOCKER_SMOKE=1 $(GO) test ./internal/executor -run TestDockerManagerPentestImageSmoke -count=1

build: build-api build-orchestrator build-executor build-migrate

build-api:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/nyx-api ./cmd/api

build-orchestrator:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/nyx-orchestrator ./cmd/orchestrator

build-executor:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/nyx-executor ./cmd/executor

build-migrate:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/nyx-migrate ./cmd/migrate

run-api:
	$(GO) run ./cmd/api

run-orchestrator:
	$(GO) run ./cmd/orchestrator

run-executor:
	$(GO) run ./cmd/executor

migrate:
	$(GO) run ./cmd/migrate

rollback:
	$(GO) run ./cmd/migrate -rollback 1

infra-up:
	docker compose -f deploy/docker-compose.local.yml up -d

infra-down:
	docker compose -f deploy/docker-compose.local.yml down

web-install:
	cd web && npm install

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

docker-build-api:
	docker build --build-arg SERVICE=api --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t nyx-api:latest .

docker-build-orchestrator:
	docker build --build-arg SERVICE=orchestrator --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t nyx-orchestrator:latest .

docker-build-executor:
	docker build --build-arg SERVICE=executor --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t nyx-executor:latest .

docker-build-executor-pentest:
	docker build -f Dockerfile.executor-pentest -t nyx-executor-pentest:latest .

docker-build-web:
	docker build -t nyx-web:latest ./web

docker-build-stack: docker-build-api docker-build-orchestrator docker-build-executor docker-build-executor-pentest docker-build-web

compose-prod-up: docker-build-executor-pentest
	docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml up --build -d

compose-prod-down:
	docker compose --env-file deploy/.env -f deploy/docker-compose.prod.yml down

vet:
	$(GO) vet ./cmd/... ./internal/...

lint: vet
	@echo "Run: golangci-lint run ./cmd/... ./internal/..."

coverage:
	$(GO) test -coverprofile=coverage.out ./cmd/... ./internal/...
	$(GO) tool cover -func=coverage.out
	@rm -f coverage.out

clean:
	rm -rf bin/ coverage.out

security-scan:
	@echo "Run: govulncheck ./..."
	@echo "Run: trivy fs ."
