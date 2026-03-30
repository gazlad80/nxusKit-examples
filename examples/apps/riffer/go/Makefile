# Makefile for riffer Go example

.PHONY: build run test clean build-generate

# Build main binary to bin/ directory (keeps git clean)
build:
	@mkdir -p bin
	go build -tags=nxuskit -v -o bin/riffer ./cmd/riffer

# Build fixture generator
build-generate:
	@mkdir -p bin
	go build -tags=nxuskit -v -o bin/generate_fixtures ./cmd/generate_fixtures

# Run the main example
run: build
	./bin/riffer

# Run tests
test:
	go test -tags=nxuskit -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	-go clean
