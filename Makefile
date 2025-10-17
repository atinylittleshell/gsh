.PHONY: build
build:
	@VERSION=$$(cat VERSION) && go build -ldflags="-X main.BUILD_VERSION=v$$VERSION" -o ./bin/gsh ./cmd/gsh/main.go

.PHONY: test
test:
	@go test -coverprofile=coverage.txt ./...

.PHONY: clean
clean:
	@rm -rf ./bin
	@rm -f coverage.out coverage.txt
