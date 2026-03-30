# Makefile for cli-assistant Go example

EXAMPLE_NAME := cli-assistant

.PHONY: build run test clean

# Build binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -tags=nxuskit -v -o bin/$(EXAMPLE_NAME) .

# Run the example
run: build
	./bin/$(EXAMPLE_NAME)

# Run tests
test:
	go test -tags=nxuskit -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
