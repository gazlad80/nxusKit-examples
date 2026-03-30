# Makefile for auth-helper Go example

EXAMPLE_NAME := auth-helper

.PHONY: build run test clean

# Build binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -tags=nxuskit -v -o bin/$(EXAMPLE_NAME) .

# Run the example (show dashboard)
run: build
	./bin/$(EXAMPLE_NAME)

# Run with provider listing
providers: build
	./bin/$(EXAMPLE_NAME) --providers

# Run tests
test:
	go test -tags=nxuskit -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
