.PHONY: build
build: generate fmt lint
	@echo "Running go build..."
	@go build -o ./bin/gsh ./cmd/gsh/main.go

.PHONY: generate
generate:
	@echo "Running go generate..."
	@go generate ./...

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
test:
	@echo "Running go test..."
	@go test -coverprofile=coverage.txt ./...

.PHONY: clean
clean:
	@rm -rf ./bin
	@rm -f coverage.out coverage.txt

.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	@brew install golangci-lint
	@brew upgrade golangci-lint
	@echo "Installing Go tools..."
	@go install golang.org/x/tools/cmd/stringer@latest
	@echo "Tools installed successfully!"
