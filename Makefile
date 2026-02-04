.PHONY: run build dev build-all build-linux build-darwin build-windows deps clean

run: build
	./cmd/beam/beam -p="tach"

build-ui:
	cd ./static/frontend && bun install && bun run build

build: deps build-ui
	go build -o ./cmd/beam/beam ./cmd/beam

dev:
	go run ./cmd/beam --dir="/Users/tacherasasi/Downloads/"

# Cross-platform builds
build-all: deps
	mkdir -p ./build
	GOOS=linux   GOARCH=amd64 go build -o ./build/beam-linux-amd64   ./cmd/beam
	GOOS=linux   GOARCH=arm64 go build -o ./build/beam-linux-arm64   ./cmd/beam
	GOOS=darwin  GOARCH=amd64 go build -o ./build/beam-darwin-amd64  ./cmd/beam
	GOOS=darwin  GOARCH=arm64 go build -o ./build/beam-darwin-arm64  ./cmd/beam
	GOOS=windows GOARCH=amd64 go build -o ./build/beam-windows-amd64.exe ./cmd/beam
	GOOS=windows GOARCH=arm64 go build -o ./build/beam-windows-arm64.exe ./cmd/beam
	cd ./build && zip beam-linux-amd64.zip beam-linux-amd64
	cd ./build && zip beam-linux-arm64.zip beam-linux-arm64
	cd ./build && zip beam-darwin-amd64.zip beam-darwin-amd64
	cd ./build && zip beam-darwin-arm64.zip beam-darwin-arm64
	cd ./build && zip beam-windows-amd64.zip beam-windows-amd64.exe
	cd ./build && zip beam-windows-arm64.zip beam-windows-arm64.exe

# Individual platform builds
build-linux: deps
	mkdir -p ./build
	GOOS=linux GOARCH=amd64 go build -o ./build/beam-linux-amd64 ./cmd/beam
	GOOS=linux GOARCH=arm64 go build -o ./build/beam-linux-arm64 ./cmd/beam
	cd ./build && zip beam-linux-amd64.zip beam-linux-amd64
	cd ./build && zip beam-linux-arm64.zip beam-linux-arm64

build-darwin: deps
	mkdir -p ./build
	GOOS=darwin GOARCH=amd64 go build -o ./build/beam-darwin-amd64 ./cmd/beam
	GOOS=darwin GOARCH=arm64 go build -o ./build/beam-darwin-arm64 ./cmd/beam
	cd ./build && zip beam-darwin-amd64.zip beam-darwin-amd64
	cd ./build && zip beam-darwin-arm64.zip beam-darwin-arm64

build-windows: deps
	mkdir -p ./build
	GOOS=windows GOARCH=amd64 go build -o ./build/beam-windows-amd64.exe ./cmd/beam
	GOOS=windows GOARCH=arm64 go build -o ./build/beam-windows-arm64.exe ./cmd/beam
	cd ./build && zip beam-windows-amd64.zip beam-windows-amd64.exe
	cd ./build && zip beam-windows-arm64.zip beam-windows-arm64.exe

deps:
	go mod tidy

clean:
	rm -f ./cmd/beam/beam
	rm -rf ./build
