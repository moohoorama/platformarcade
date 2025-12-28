.PHONY: build run wasm serve clean test

# Binary name
BINARY_NAME=mg

# Build native binary
build:
	go build -o bin/$(BINARY_NAME) ./cmd/game

# Run the game
run: build
	cd bin && ./$(BINARY_NAME)

# Run directly without building to bin
dev:
	go run ./cmd/game

# Build WebAssembly
wasm:
	GOOS=js GOARCH=wasm go build -o web/game.wasm ./cmd/game
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/

# Serve WebAssembly locally
serve: wasm
	@echo "Starting server at http://localhost:8080"
	cd web && python3 -m http.server 8080

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -v -cover ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f web/game.wasm
	rm -f web/wasm_exec.js

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build for all platforms
build-all: build wasm
	@echo "Build complete: bin/$(BINARY_NAME) and web/game.wasm"
