# Makefile for bn-structure-learning Go example

EXAMPLE_NAME := bn-structure-learning

.PHONY: build run test clean

# Build binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -tags nxuskit -v -o bin/$(EXAMPLE_NAME) .

# Run the example (default scenario: golf)
run: build
	./bin/$(EXAMPLE_NAME) --scenario golf

# Run tests
test:
	go test -tags nxuskit -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
