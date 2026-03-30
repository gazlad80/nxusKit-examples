# Makefile for bayesian-inference Go example

EXAMPLE_NAME := bayesian-inference

.PHONY: build run test clean

# Build binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -tags nxuskit -v -o bin/$(EXAMPLE_NAME) .

# Run the example (default scenario: haunted-house)
run: build
	./bin/$(EXAMPLE_NAME) --scenario haunted-house

# Run tests
test:
	go test -tags nxuskit -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
