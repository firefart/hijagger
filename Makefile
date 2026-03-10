.DEFAULT_GOAL := build

.PHONY: update
update:
	go get -u
	go mod tidy

.PHONY: build
build: test
	go fmt ./...
	go vet ./...
	go build

.PHONY: run
run: build
	./hijagger

.PHONY: lint
lint:
	"$$(go env GOPATH)/bin/golangci-lint" run ./...
	go mod tidy

.PHONY: lint-update
lint-update:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: test
test:
	go test -race -cover ./...
