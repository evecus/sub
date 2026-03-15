.PHONY: all build build-web build-go clean install release

BINARY := sub-store
ifeq ($(OS),Windows_NT)
  BINARY := sub-store.exe
endif

all: build

# Install frontend deps (pnpm preferred, falls back to npm)
install:
	cd web && (pnpm install 2>/dev/null || npm install)

# Build frontend → web/dist
build-web:
	cd web && (pnpm build 2>/dev/null || npm run build)

# Build Go binary (embeds web/dist)
build-go:
	go mod tidy
	go build -ldflags="-s -w" -o $(BINARY) .

# Full build
build: install build-web build-go
	@echo ""
	@echo "✅ Build complete: ./$(BINARY)"
	@echo "   Run: ./$(BINARY)"
	@echo "   Open: http://localhost:8080"

# Rebuild without reinstalling deps
rebuild: build-web build-go
	@echo "✅ Rebuilt: ./$(BINARY)"

# Clean
clean:
	rm -f $(BINARY)
	rm -rf web/dist
	rm -rf web/node_modules

# Cross-compile
release: build-web
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64  go build -ldflags="-s -w" -o dist/sub-store-linux-amd64 .
	GOOS=linux   GOARCH=arm64  go build -ldflags="-s -w" -o dist/sub-store-linux-arm64 .
	GOOS=linux   GOARCH=arm    go build -ldflags="-s -w" -o dist/sub-store-linux-arm .
	GOOS=darwin  GOARCH=amd64  go build -ldflags="-s -w" -o dist/sub-store-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64  go build -ldflags="-s -w" -o dist/sub-store-darwin-arm64 .
	GOOS=windows GOARCH=amd64  go build -ldflags="-s -w" -o dist/sub-store-windows-amd64.exe .
	@echo "✅ Release binaries in ./dist/"
