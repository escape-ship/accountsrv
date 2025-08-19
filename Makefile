all: build

init:
	@echo "Initializing..."
	@$(MAKE) tool_download
	@$(MAKE) build

build:
	@echo "Building..."
	@go mod tidy
	@go mod download
	@$(MAKE) sqlc_gen
	@go build -o bin/$(shell basename $(PWD)) ./cmd

build_alone:
	@go build -o bin/$(shell basename $(PWD)) ./cmd

docker:
	@docker build -t ghcr.io/escape-ship/accountsrv:latest .

pushall:
	@docker build -t ghcr.io/escape-ship/accountsrv:latest .
	@docker push ghcr.io/escape-ship/accountsrv:latest

sqlc_gen:
	@echo "Generating sqlc..."
	@cd internal/infra/sqlc && \
	sqlc generate

tool_download:
	$(MAKE) sqlc_download

sqlc_download:
	@echo "Downloading sqlc..."
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

run:
	@echo "Running..."
	@./bin/$(shell basename $(PWD))

linter-golangci: ### check by golangci linter
	golangci-lint run
.PHONY: linter-golangci

clean:
	rm -f bin/$(shell basename $(PWD))