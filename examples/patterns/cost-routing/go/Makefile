# Makefile for cost-routing Go example

EXAMPLE_NAME := cost-routing

.PHONY: build run test clean

# Build binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -v -o bin/$(EXAMPLE_NAME) .

# Run the example
run: build
	./bin/$(EXAMPLE_NAME)

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
