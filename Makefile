BINARY=engine
COMPOSE=docker-compose
MIGRATION_PATH=migrations/postgres

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_.-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help

test: generate generate.mock ## Run tests
	GO_ENV=test go test -v -cover -covermode=atomic ./...
.PHONY: test

coverage: generate generate.mock ## Run tests with coverage
	GO_ENV=test go test $(shell go list ./... |  grep -v  mocks) -coverprofile=tmp/cover.out  -coverpkg=./...
.PHONY: coverage

coverage.view: ## View coverage report
	go tool cover -html=tmp/cover.out
.PHONY: coverage.view

coverage.check: ## Check coverage
	go tool cover -func tmp/cover.out
.PHONY: coverage.check

dev: setup.githook generate ## Run the application in development mode
	go run github.com/air-verse/air
.PHONY: dev

run: generate ## Run the application
	go run ./cmd/app
.PHONY: run

build: generate ## Build the application
	go build -ldflags '-s -w' -o ${BINARY} ./cmd/app/main.go
.PHONY: build

clean: ## Clean up the project
	@if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	@find . -name *mock* -delete
	@rm -rf docs/ ./di/wire_gen.go tmp/cover.out
.PHONY: clean

docker.build: ## Build the docker image
	$(COMPOSE) build
.PHONY: docker.build

docker.start: ## Start the docker container
	$(COMPOSE) compose up -d
.PHONY: docker.start

docker.stop: ## Stop the docker container
	$(COMPOSE) compose down
.PHONY: docker.stop

lint.prepare: ## Prepare the environment for linting
	@echo "Installing golangci-lint"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.64.8
.PHONY: lint.prepare

lint: ## Run linters
	golangci-lint run ./... --fix
.PHONY: lint

generate: ## Generate code
	go generate -skip="mockgen" ./...
.PHONY: generate

generate.mock: ## Generate mock code
	go generate -run="mockgen" ./...
.PHONY: generate.mock

migrate.up: ## Run database migrations up
	go run cmd/migrate/main.go up
.PHONY: migrate.up

migrate.down: ## Run database migrations down
	go run cmd/migrate/main.go down
.PHONY: migrate.down

migrate.step-up: ## Run database migrations step up
	go run cmd/migrate/main.go step-up
.PHONY: migrate.step-up

migrate.drop: ## Drop all database migrations
	go run cmd/migrate/main.go drop
.PHONY: migrate.drop

migrate.create: ## Create a new migration file. Usage: make migrate.create name=<migration name>
	@if [ -z "$(name)" ]; then \
		echo "Please set the name variable"; \
		echo "Example: make migrate.create name=add_users_table"; \
		exit 1; \
	fi
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@latest create -ext sql -dir $(MIGRATION_PATH) -seq $(name)
.PHONY: migrate.create

migrate.order: ## Order migrations. Usage: make migrate-order table=<table name> n=<order number>
	cmd/migrate/order.sh $(table) $(n)
.PHONY: migrate.order

setup.githook: ## Setup git hooks
	git config core.hooksPath .githooks
.PHONY: setup.githook

domains: ## Create simple empty domains file. Usage: make domains name=<domains name>
	@if [ -z "$(name)" ]; then \
    	echo "Please set the name variable"; \
    	echo "Example: make domains name=service"; \
    	exit 1; \
    fi
	@mkdir -p ./internal/domains/$(name)/model/dto
	@echo "package model" > ./internal/domains/$(name)/model/model.go
	@echo "package dto" > ./internal/domains/$(name)/model/dto/dto.go
	@mkdir -p ./internal/domains/$(name)/repository
	@echo "package repository" > ./internal/domains/$(name)/repository/repository.go
	@mkdir -p ./internal/domains/$(name)/service
	@echo "package service" > ./internal/domains/$(name)/service/service.go
	@echo "Domain $(name) created successfully"
.PHONY: domains
