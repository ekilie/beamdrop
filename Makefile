.PHONY: run build dev build-all build-linux build-darwin build-windows deps clean build-ui release checksums

# ── Variables ──────────────────────────────────────────────────────────────────
APP_NAME    := beamdrop
MODULE      := github.com/tachRoutine/beamdrop-go
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -s -w \
	-X '$(MODULE)/config.VERSION=$(VERSION)' \
	-X '$(MODULE)/config.Commit=$(COMMIT)' \
	-X '$(MODULE)/config.BuildDate=$(BUILD_DATE)'
BUILD_DIR   := ./build
CGO_ENABLED ?= 0

# ── Development ───────────────────────────────────────────────────────────────
run: build
	./cmd/beam/beamdrop -p="tach"

build-ui:
	cd ./static/frontend && bun install && bun run build

build: deps build-ui
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o ./cmd/beam/beamdrop ./cmd/beam

dev:
	go run ./cmd/beam --dir="."

# ── Cross-platform builds (all) ──────────────────────────────────────────────
build-all: deps build-ui
	@echo "==> Building for all platforms ($(VERSION))..."
	mkdir -p $(BUILD_DIR)
	# macOS Apple Silicon (M1/M2/M3/M4)
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64      ./cmd/beam
	# macOS Intel
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64      ./cmd/beam
	# Linux amd64
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64       ./cmd/beam
	# Linux arm64
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64       ./cmd/beam
	# Windows amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/beam
	# Windows arm64
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe ./cmd/beam
	@echo "==> Packaging archives..."
	cd $(BUILD_DIR) && cp $(APP_NAME)-darwin-arm64 $(APP_NAME) && tar czf $(APP_NAME)-darwin-arm64.tar.gz  $(APP_NAME) && rm $(APP_NAME)
	cd $(BUILD_DIR) && cp $(APP_NAME)-darwin-amd64 $(APP_NAME) && tar czf $(APP_NAME)-darwin-amd64.tar.gz  $(APP_NAME) && rm $(APP_NAME)
	cd $(BUILD_DIR) && cp $(APP_NAME)-linux-amd64  $(APP_NAME) && tar czf $(APP_NAME)-linux-amd64.tar.gz   $(APP_NAME) && rm $(APP_NAME)
	cd $(BUILD_DIR) && cp $(APP_NAME)-linux-arm64  $(APP_NAME) && tar czf $(APP_NAME)-linux-arm64.tar.gz   $(APP_NAME) && rm $(APP_NAME)
	cd $(BUILD_DIR) && cp $(APP_NAME)-windows-amd64.exe $(APP_NAME).exe && zip $(APP_NAME)-windows-amd64.zip $(APP_NAME).exe && rm $(APP_NAME).exe
	cd $(BUILD_DIR) && cp $(APP_NAME)-windows-arm64.exe $(APP_NAME).exe && zip $(APP_NAME)-windows-arm64.zip $(APP_NAME).exe && rm $(APP_NAME).exe
	@echo "==> Done! Binaries in $(BUILD_DIR)/"

# ── Individual platform builds ───────────────────────────────────────────────
build-darwin: deps build-ui
	@echo "==> Building for macOS..."
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/beam
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/beam
	cd $(BUILD_DIR) && cp $(APP_NAME)-darwin-arm64 $(APP_NAME) && tar czf $(APP_NAME)-darwin-arm64.tar.gz $(APP_NAME) && rm $(APP_NAME)
	cd $(BUILD_DIR) && cp $(APP_NAME)-darwin-amd64 $(APP_NAME) && tar czf $(APP_NAME)-darwin-amd64.tar.gz $(APP_NAME) && rm $(APP_NAME)

build-linux: deps build-ui
	@echo "==> Building for Linux..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/beam
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/beam
	cd $(BUILD_DIR) && cp $(APP_NAME)-linux-amd64 $(APP_NAME) && tar czf $(APP_NAME)-linux-amd64.tar.gz $(APP_NAME) && rm $(APP_NAME)
	cd $(BUILD_DIR) && cp $(APP_NAME)-linux-arm64 $(APP_NAME) && tar czf $(APP_NAME)-linux-arm64.tar.gz $(APP_NAME) && rm $(APP_NAME)

build-windows: deps build-ui
	@echo "==> Building for Windows..."
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/beam
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe ./cmd/beam
	cd $(BUILD_DIR) && cp $(APP_NAME)-windows-amd64.exe $(APP_NAME).exe && zip $(APP_NAME)-windows-amd64.zip $(APP_NAME).exe && rm $(APP_NAME).exe
	cd $(BUILD_DIR) && cp $(APP_NAME)-windows-arm64.exe $(APP_NAME).exe && zip $(APP_NAME)-windows-arm64.zip $(APP_NAME).exe && rm $(APP_NAME).exe

# ── SHA256 checksums ─────────────────────────────────────────────────────────
checksums:
	@echo "==> Generating checksums..."
	cd $(BUILD_DIR) && shasum -a 256 *.tar.gz *.zip > checksums.txt
	@cat $(BUILD_DIR)/checksums.txt

# ── GitHub Release (requires gh CLI) ─────────────────────────────────────────
release: clean build-all checksums
	@echo "==> Creating GitHub release $(VERSION)..."
	gh release create $(VERSION) \
		--title "Beamdrop $(VERSION)" \
		--generate-notes \
		$(BUILD_DIR)/*.tar.gz \
		$(BUILD_DIR)/*.zip \
		$(BUILD_DIR)/checksums.txt

# ── Utilities ─────────────────────────────────────────────────────────────────
deps:
	go mod tidy

clean:
	rm -f ./cmd/beam/beamdrop
	rm -rf $(BUILD_DIR)
