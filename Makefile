.PHONY: build test

build:
	go build ./...

test: bin/sqlc-gen-go.wasm
	go test ./...

all: bin/sqlc-gen-go bin/sqlc-gen-go.wasm

# auto: use go.mod toolchain (download if newer than local); local breaks when go.mod > installed Go.
bin/sqlc-gen-go: bin go.mod go.sum $(wildcard **/*.go)
	cd plugin && GOTOOLCHAIN=auto go build -o ../bin/sqlc-gen-go ./main.go

bin/sqlc-gen-go.wasm: bin/sqlc-gen-go
	cd plugin && GOTOOLCHAIN=auto GOOS=wasip1 GOARCH=wasm go build -o ../bin/sqlc-gen-go.wasm main.go

bin:
	mkdir -p bin