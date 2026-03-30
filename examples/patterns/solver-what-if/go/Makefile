# Makefile for solver-what-if Go example

EXAMPLE_NAME := solver-what-if

.PHONY: build run test clean

# Build binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -tags nxuskit -v -o bin/$(EXAMPLE_NAME) .

# Run the example (default scenario: wedding)
run: build
	./bin/$(EXAMPLE_NAME) --scenario wedding

# Run tests
test:
	go test -tags nxuskit -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
