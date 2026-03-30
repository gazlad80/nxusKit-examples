# Makefile for bn-solver-clips-pipeline Go example

EXAMPLE_NAME := bn-solver-clips-pipeline

.PHONY: build run test clean

# Build binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -tags nxuskit -v -o bin/$(EXAMPLE_NAME) .

# Run the example (default scenario: festival)
run: build
	./bin/$(EXAMPLE_NAME) --scenario festival

# Run tests
test:
	go test -tags nxuskit -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
