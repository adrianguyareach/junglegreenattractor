.PHONY: test test-race build

# Run all tests in all packages (recursive).
test:
	go test ./...

# Same as test, with the race detector enabled.
test-race:
	go test ./... -race

build:
	go build -o junglegreenattractor ./cmd/junglegreenattractor/
	go build -o jga ./cmd/jga/
