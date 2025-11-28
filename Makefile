.PHONY: all
all: ci

.PHONY: build
build:
	@VERSION=$$(cat VERSION) && go build -ldflags="-X main.BUILD_VERSION=v$$VERSION" -o ./bin/gsh ./cmd/gsh/main.go

.PHONY: test
test:
	@go test -coverprofile=coverage.txt ./...

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

.PHONY: vulncheck
vulncheck:
	@echo "Running govulncheck..."
	@govulncheck ./...

.PHONY: ci
ci: lint vulncheck test build

.PHONY: tools
tools:
	@echo "Installing tools..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: clean
clean:
	@rm -rf ./bin
	@rm -f coverage.out coverage.txt
