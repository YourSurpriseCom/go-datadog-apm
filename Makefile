.DEFAULT_GOAL := help

.PHONY: help
help:
	@printf "\033[33mUsage:\033[0m\n  make [target] [arg=\"val\"...]\n\n\033[33mTargets:\033[0m\n"
	@grep -E '^[-a-zA-Z0-9_\.\/]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[32m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: install 
install: ## install dependencies 
	@go mod download

.PHONY: check
check: golangci-lint go-vet unit-tests security-code-scan security-vulnerability-scan ## Run application checks

.PHONY: golangci-lint
golangci-lint:
	@golangci-lint run

.PHONY: go-vat
go-vet:
	@go vet ./...

.PHONY: unit-tests
unit-tests: ## run unit tests
	@go test -v ./...

.PHONY: security-code-scan
security-code-scan:
	@gosec ./...

.PHONY: security-vulnerability-scan
security-vulnerability-scan:
	@govulncheck -show verbose ./...

.PHONY: coverage
coverage:  ## show coverage
	go test -failfast -coverprofile=coverage.out ./... 
	go tool cover -html="coverage.out"
