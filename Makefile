.PHONY: build
build: generate fmt lint
	@echo "Running go build..."
	@go build -o ./bin/gsh ./cmd/gsh/main.go

.PHONY: generate
generate:
	@echo "Running go generate..."
	@go generate ./...

.PHONY: check-generate
check-generate: generate
	@echo "Checking if generated files are up to date..."
	@if [ -n "$$(git status --porcelain | grep '_string.go')" ]; then \
		echo "Error: Generated files are out of date. Please run 'make generate'"; \
		git status --porcelain | grep '_string.go'; \
		exit 1; \
	fi
	@echo "All generated files are up to date!"

.PHONY: fmt
fmt:
	@echo "Running golangci-lint fmt..."
	@golangci-lint fmt

.PHONY: lint
lint:
	@echo "Running golangci-lint run..."
	@golangci-lint run

.PHONY: lint-fix
lint-fix:
	@echo "Running golangci-lint run with auto-fix..."
	@golangci-lint run --fix

.PHONY: test
test: generate
	@echo "Running go test..."
	@go test -coverprofile=coverage.txt ./...

.PHONY: clean
clean:
	@rm -rf ./bin
	@rm -f coverage.out coverage.txt
	@echo "Cleaning generated files..."
	@find . -name '*_string.go' -type f -delete

.PHONY: ci
ci: check-generate fmt lint test
	@echo "CI checks passed!"

.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	@echo "Installing Go tools..."
	@go install golang.org/x/tools/cmd/stringer@latest
	@if command -v brew >/dev/null 2>&1; then \
		echo "Installing golangci-lint via brew..."; \
		brew install golangci-lint || brew upgrade golangci-lint; \
	else \
		echo "Brew not found, skipping golangci-lint (use golangci-lint-action in CI or install manually)"; \
	fi
	@echo "Tools installed successfully!"
