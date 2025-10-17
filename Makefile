.PHONY: build
build:
	@go build -o ./bin/gsh ./cmd/gsh/main.go

.PHONY: test
test:
	@go test -coverprofile=coverage.txt ./...

.PHONY: clean
clean:
	@rm -rf ./bin
	@rm -f coverage.out coverage.txt
