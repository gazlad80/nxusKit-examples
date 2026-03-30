.PHONY: build test clean

build:
	mkdir -p bin && go build -tags nxuskit -o bin/llm-solver-hybrid .

test: build
	@echo "Run: bin/llm-solver-hybrid --scenario seating"

clean:
	rm -rf bin/
